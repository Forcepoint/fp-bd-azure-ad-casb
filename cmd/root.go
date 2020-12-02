package cmd

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net/http"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var (
	cfgFile          string
	RiskCoreInstance *RiskScore
	AzureCliInstance *AzureCLI
)

var rootCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the riskScore exporter and manager",
	Long:  ``,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file azure_casb.yml)")
	if err := rootCmd.MarkPersistentFlagRequired("config"); err != nil {
		logrus.Fatal(err)
	}
}

func initConfig() {
	viper.SetDefault("CASB_USER_NAME", "")
	viper.SetDefault("CASB_PASSWORD", "")
	viper.SetDefault("RISK_SCORE_URL", "")
	viper.SetDefault("AZURE_ADMIN_LOGIN_NAME", "")
	viper.SetDefault("AZURE_ADMIN_LOGIN_PASSWORD", "")
	viper.SetDefault("mail-nickname", false)
	viper.SetDefault("AZURE_GROUPS_NAME", "")
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(home)
		viper.SetConfigName("azure_casb")
	}

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err == nil {
		viper.WatchConfig()
		viper.OnConfigChange(func(e fsnotify.Event) {
			if viper.GetBool("LOGGER_JSON_FORMAT") {
				logrus.SetFormatter(&logrus.JSONFormatter{})
			} else {
				logrus.SetFormatter(&logrus.TextFormatter{})
			}

		})
	}
	RiskCoreInstance = &RiskScore{
		UserName:     viper.GetString("CASB_USER_NAME"),
		Password:     viper.GetString("CASB_PASSWORD"),
		Client:       &http.Client{},
		RiskScoreUrl: viper.GetString("RISK_SCORE_URL"),
	}
	AzureCliInstance = &AzureCLI{}
}
