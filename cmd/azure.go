package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	errorWrapper "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

var (
	ErrBadFormatEmail = errors.New("invalid email format")
	regexPEmail       = `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`
)

type AzureCLI struct {
	IsLogin bool
}

// is azure CLi instance on your machine has already active session
func (a *AzureCLI) IsAlreadyLogin() (string, bool) {
	c := "az account show --query user.name -o tsv"
	output, err := ExecuteCmd(c)
	if err != nil {
		if strings.Contains(err.Error(), "Please run 'az login' to setup account") {
			a.IsLogin = false
			return "", false
		}
	}
	a.IsLogin = true
	return strings.TrimSpace(output), true
}

// login to azure
func (a *AzureCLI) Login() error {
	_, isLogin := a.IsAlreadyLogin()

	if !isLogin {
		var stdout, stderr bytes.Buffer
		if viper.GetString("AZURE_ADMIN_LOGIN_NAME") == "" {
			username, err := ReadUserLoginName()
			if err != nil {
				logrus.Fatal(err)
			}
			viper.Set("AZURE_ADMIN_LOGIN_NAME", username)
		}
		if viper.GetString("AZURE_ADMIN_LOGIN_PASSWORD") == "" {
			fmt.Printf("Enter password for '%s' and press Enter: ",
				viper.GetString("AZURE_ADMIN_LOGIN_NAME"))
			bytePassword, err := terminal.ReadPassword(syscall.Stdin)
			if err != nil {
				return err
			}
			password := string(bytePassword)
			fmt.Println() // do not remove it
			if len(password) == 0 {
				return errors.New("please enter a valid password")
			}
			viper.Set("AZURE_ADMIN_LOGIN_PASSWORD", strings.TrimSpace(password))
		}
		//login to azure
		c1 := fmt.Sprintf("az login -u %s -p '%s'",
			viper.GetString("AZURE_ADMIN_LOGIN_NAME"),
			viper.GetString("AZURE_ADMIN_LOGIN_PASSWORD"))
		exe := exec.Command("sh", "-c", c1)
		exe.Stderr = &stderr
		exe.Stdout = &stdout
		err := exe.Run()
		errorResult := string(stderr.Bytes())
		if len(errorResult) != 0 {
			if strings.Contains(errorResult, "Error validating credentials due to invalid username or password") {
				return errors.New("error in validating credentials due to invalid username or password")
			}
			return errors.New(errorResult)
		}
		if err != nil {
			return errors.New("failed in executing the azure login command")
		}
		a.IsLogin = true
	}
	return nil
}

// azure logout
func (a *AzureCLI) Logout() error {
	exe := exec.Command("sh", "-c", "az logout")
	err := exe.Run()
	if err != nil {
		err = errorWrapper.Wrap(err, "Failed in executing the azure logout command")
		return err
	}
	a.IsLogin = false
	return nil
}

//execute a azure CLI command
func ExecuteCmd(cmd string) (string, error) {
	var stdout, stderr bytes.Buffer
	exe := exec.Command("sh", "-c", cmd)
	exe.Stderr = &stderr
	exe.Stdout = &stdout
	err := exe.Run()
	errorResult := string(stderr.Bytes())
	if len(errorResult) != 0 && !strings.Contains(errorResult, "deprecated") {
		return "", errors.New(errorResult)
	}
	if err != nil && !strings.Contains(errorResult, "deprecated") {
		return "", errors.New(fmt.Sprintf("failed in executing the azure command: %s", cmd))
	}
	output := string(stdout.Bytes())
	if len(output) != 0 {
		return output, nil
	}
	return "", nil
}

// the user name which has an active azure CLI session
func ReadUserLoginName() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter your Azure administrator's username: ")
		text, err := reader.ReadString('\n')
		if err != nil {
			return "", errorWrapper.Wrap(err, "failed in reading username from console")
		}
		text = strings.Replace(text, "\n", "", -1)
		isValidEmail, err := ValidateEmailAddress(text)
		if !isValidEmail {
			logrus.Error(err)
		} else {
			return text, nil
		}
	}
}

// validate if the format of a given email is correct
func ValidateEmailAddress(email string) (bool, error) {
	emailRegexp := regexp.MustCompile(regexPEmail)
	if !emailRegexp.MatchString(email) {
		return false, errorWrapper.Wrap(ErrBadFormatEmail, email)
	}
	return true, nil
}

// get all users in your azure active directory
func (a *AzureCLI) GetAllUsers(nickName bool) ([]string, error) {
	filter := "userPrincipalName"
	if nickName {
		filter = "mailNickname"
	}
	c := fmt.Sprintf("az ad user list --query [].%s -o tsv", filter)
	output, err := ExecuteCmd(c)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(output), "\n"), nil
}

//map each risk score to a risk group. the mapping info must be defined in the config file
func (a *AzureCLI) MapRIskScoreToGroup(users map[string]int) (map[string]string, error) {
	UserRiskGroup := make(map[string]string)
	riskScoreRanges := viper.Get("MAP_RISK_SCORE")
	riskScoreRangesList := riskScoreRanges.([]interface{})
	for _, riskScores := range riskScoreRangesList {
		riskScoreMap := riskScores.(map[interface{}]interface{})
		for k, v := range riskScoreMap {
			maxValue := 0
			toValue := 0
			fromValue := 0
			if strings.Contains(k.(string), "+") {
				value := strings.ReplaceAll(k.(string), "+", "")
				valueInt, err := strconv.Atoi(value)
				if err != nil {
					return nil, errorWrapper.Wrap(err, "failed in converting string to int")
				}
				maxValue = valueInt
			} else {
				keyParts := strings.Split(k.(string), "-")
				if len(keyParts) != 2 {
					return nil, errors.New("format of mapping riskScore to a risk group is not correct in the config file: " + k.(string))
				}
				from, err := strconv.Atoi(keyParts[0])
				if err != nil {
					return nil, errorWrapper.Wrap(err, "failed in mapping riskScore range to a risk level group")
				}
				to, err := strconv.Atoi(keyParts[1])
				if err != nil {
					return nil, errorWrapper.Wrap(err, "failed in mapping riskScore range to a risk level group")
				}
				toValue = to
				fromValue = from
			}
			for user, score := range users {
				if score >= maxValue {
					UserRiskGroup[user] = v.(string)
				} else if score >= fromValue && score <= toValue {
					UserRiskGroup[user] = v.(string)
				}
			}
		}
	}
	return UserRiskGroup, nil
}

//get user's groups
func (a *AzureCLI) GetUserGroups(userName string) (string, []string, error) {
	c := ""
	if viper.GetBool("mail-nickname") {
		parts := strings.Split(userName, "@")
		userName = parts[0]
		c = fmt.Sprintf("az ad user list --query \"[?contains(mailNickname,'%s')].objectId\" -o tsv", userName)
	} else {
		c = fmt.Sprintf("az ad user show --id '%s' --query objectId -o tsv", userName)
	}
	userId, err := ExecuteCmd(c)
	if err != nil {
		return "", nil, err
	}
	userId = strings.TrimSpace(userId)
	c = fmt.Sprintf("az ad user get-member-groups --id %s --query [].displayName -o tsv", userId)
	output, err := ExecuteCmd(c)
	if err != nil {
		return "", nil, err
	}
	output = strings.TrimSpace(output)
	return userId, strings.Split(output, "\n"), nil
}

// update user's login policy in azure
func (a *AzureCLI) ProcessOneUser(user string, group string, azureGroups []string) error {
	userId, existsGroups, err := a.GetUserGroups(user)
	if err != nil {
		return err
	}
	if !ElementInList(existsGroups, group) {
		if err := a.CleanUserGroups(user, userId, existsGroups, azureGroups); err != nil {
			return err
		}
		if err := a.AddUserToGroup(user, userId, group); err != nil {
			return err
		}
	}
	return nil
}

// remove a user from all risk groups
func (a *AzureCLI) CleanUserGroups(user string, userId string, existsGroups []string, azureGroups []string) error {
	var subtractionGroups []string
	for _, group := range existsGroups {
		if ElementInList(azureGroups, group) {
			subtractionGroups = append(subtractionGroups, group)
		}
	}
	if len(subtractionGroups) == 0 {
		return nil
	}
	for _, group := range subtractionGroups {
		if err := a.RemoveUserGroup(userId, group); err != nil {
			return err
		}
		logrus.Infof("Removed user:%s from previous risk-level group:%s", user, group)
	}
	return nil
}

// add a user to a risk group
func (a *AzureCLI) RemoveUserGroup(userId string, group string) error {
	groupId, err := a.GetGroupId(group)
	if err != nil {
		return err
	}
	c := fmt.Sprintf("az ad group member remove -g %s --member-id %s", groupId, userId)
	_, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	return nil
}

//get user's ObjectId
func (a *AzureCLI) GetUserId(user string) (string, error) {
	c := fmt.Sprintf("az ad user show --id '%s' --query objectId -o tsv", user)
	output, err := ExecuteCmd(c)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// get group's ObjectId
func (a *AzureCLI) GetGroupId(group string) (string, error) {
	c := fmt.Sprintf("az ad group show -g '%s' --query objectId -o tsv", group)
	output, err := ExecuteCmd(c)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

//add a user to a group
func (a *AzureCLI) AddUserToGroup(user string, userId string, group string) error {
	groupId, err := a.GetGroupId(group)
	if err != nil {
		return err
	}
	c := fmt.Sprintf("az ad group member add -g '%s' --member-id %s", groupId, userId)
	_, err = ExecuteCmd(c)
	if err != nil {
		return err
	}
	logrus.Infof("Added user:%s to new risk-level group:%s", user, group)
	if viper.GetBool("TERMINATE_USER_ACTIVE_SESSION") {
		if err := a.TerminateSession(userId); err != nil {
			return err
		}
		logrus.Warningf("All active session for user %s has been terminated", user)
	}
	return nil
}

func (a *AzureCLI) TerminateSession(userId string) error {
	c := fmt.Sprintf("az rest --method POST --uri 'https://graph.microsoft.com/v1.0/users/%s/revokeSignInSessions'", userId)
	_, err := ExecuteCmd(c)
	return err
}
