package cmd

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"time"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run RiskScore exporter and manager",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if err := AzureCliInstance.Login(); err != nil {
			logrus.Fatal(err)
		}
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)
		defer close(signalChan)
		go func() {
			select {
			case <-signalChan:
				fmt.Println()
				os.Exit(0)
			}
		}()

		for {
			accounts, accountLoginNames, err := RiskCoreInstance.ParseRiskScore()
			if err != nil {
				logrus.Error(err)
			}
			if err := RiskCoreInstance.ProcessRiskScores(accounts, accountLoginNames); err != nil {
				logrus.Error(err)
			}
			time.Sleep(time.Duration(viper.GetInt("RISK_MANAGER_INTERVAL_TIME")) * time.Minute)
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().BoolP("mail-nickname", "m", false, "Compare Forcepoint CASB user's MailNickName with azure user's MailNickName")
	if err := viper.BindPFlag("mail-nickname", runCmd.Flags().Lookup("mail-nickname")); err != nil {
		logrus.Fatal(err)
	}
}
