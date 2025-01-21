/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/sepiroth887/goyoming-handler/handler"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "goyoming-handler",
	Short: "wyoming event handler. Acts as voice start/stop notifier and synteziser to HA mediaplay or TTS",
	Long:  ``,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}

		var config handler.Configuration
		cData, err := os.ReadFile(viper.GetString("config"))
		if err != nil {
			log.Errorf("failed to load config: %v", err)
			os.Exit(1)
		}

		err = yaml.Unmarshal(cData, &config)
		if err != nil {
			log.Errorf("failed to parse config: %v", err)
			os.Exit(2)
		}
		log.Info("Starting up handler")
		h := handler.New(config)

		log.Infof("listening on %s:%d", config.Listen, config.Port)
		err = h.ListenAndServe()
		if err != nil {
			log.Error(err)
			os.Exit(3)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.goyoming-handler.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().StringP("config", "c", "./config.yml", "goyoming config yaml file")
	rootCmd.Flags().BoolP("debug", "v", false, "Debug logging")
	viper.BindPFlags(rootCmd.Flags())
}
