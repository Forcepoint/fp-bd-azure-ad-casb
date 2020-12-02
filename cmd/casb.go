package cmd

import (
	"errors"
	_ "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type RiskScore struct {
	UserName     string
	Password     string
	RiskScoreUrl string
	*http.Client
}

//override the http.Client.Do function
func (c *RiskScore) Do(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(c.UserName, c.Password)
	return http.DefaultClient.Do(req)
}

//get the risk scores from Forcepoint CASB
func (c *RiskScore) GetRiskScores() (string, error) {
	req, err := http.NewRequest("GET", c.RiskScoreUrl, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New("got an empty response from CASB instance")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return string(body), nil
}

// parse the downloaded risk score csv file
func (c *RiskScore) ParseRiskScore() (map[string]int, map[string][]string, error) {
	var accounts = make(map[string]int)
	var accountsLoginNames = make(map[string][]string)
	output, err := c.GetRiskScores()
	if err != nil {
		return nil, nil, err
	}
	if strings.Contains(output, "<html>") {
		return nil, nil, errors.New("failed in login to Forcepoint CASB in order to download riskScore.csv")
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	lines = lines[1:]
	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) >= 10 {
			account, loginName, riskScore := parts[0], parts[1], parts[2]
			if !KeyInMap(accounts, account) {
				accounts[account] = 0
				accountsLoginNames[account] = []string{}
			}
			riskScoreFloat, _ := strconv.ParseFloat(riskScore, 32)
			scoreInt := int(riskScoreFloat)
			if accounts[account] < scoreInt {
				accounts[account] = scoreInt
			}
			if !ElementInList(accountsLoginNames[account], loginName) {
				accountsLoginNames[account] = append(accountsLoginNames[account], loginName)
			}
		}
	}
	return accounts, accountsLoginNames, nil
}

// is element exists in a list
func ElementInList(l []string, e string) bool {
	for _, v := range l {
		if v == e {
			return true
		}
	}
	return false
}

//is element exists in a map
func KeyInMap(m map[string]int, key string) bool {
	for k, _ := range m {
		if k == key {
			return true
		}
	}
	return false
}

// process the risk score for each exist user
func (c *RiskScore) ProcessRiskScores(accounts map[string]int, accountsLoginNames map[string][]string) error {
	//is user is an azure user
	var foundUsers = make(map[string]int)
	users, err := AzureCliInstance.GetAllUsers(viper.GetBool("mail-nickname"))

	if err != nil {
		return err
	}
	for k, v := range accountsLoginNames {
		for _, name := range v {
			nickName := name
			if viper.GetBool("mail-nickname") {
				nameParts := strings.Split(name, "@")
				nickName = nameParts[0]
			}
			if ElementInList(users, nickName) {
				foundUsers[name] = accounts[k]
			}
		}
	}
	userNewGroup, err := AzureCliInstance.MapRIskScoreToGroup(foundUsers)
	if err != nil {
		logrus.Error(err)
	}
	var azureGroupsName []string
	if viper.GetString("AZURE_GROUPS_NAME") == "" {
		return errors.New("AZURE_GROUPS_NAME parameter is missing in your config file")
	}
	azureGroupsName = strings.Split(viper.GetString("AZURE_GROUPS_NAME"), ",")
	for user, group := range userNewGroup {
		if err := AzureCliInstance.ProcessOneUser(user, group, azureGroupsName); err != nil {
			return err
		}
	}
	return nil
}
