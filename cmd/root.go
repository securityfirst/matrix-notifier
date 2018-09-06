package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/kelseyhightower/envconfig"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	conf    *config
	logger  *log.Logger
)

// RootCmd is the main command.
var RootCmd = &cobra.Command{
	Use:   "matrix-notifier",
	Short: "Matrix server for custom notifications",
	Long:  `A web server that uses matrix user system to handle organisations and notifications.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.a.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	logger = log.New(os.Stdout, "[matrix-notifier] ", log.Ltime|log.Lshortfile)
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".matrix-notifier")
	}
	viper.ReadInConfig()
	if file := viper.ConfigFileUsed(); file != "" {
		logger.Println("Using config file:", file)
		if err := viper.Unmarshal(&conf); err != nil {
			logger.Fatalln("Invalid configuration:", err)
		}
		conf.Init()
	}
	if err := envconfig.Process(".matrix-notifier", conf); err != nil {
		logger.Fatalln("Env:", err)
	}
}
