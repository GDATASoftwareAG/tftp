package main

import (
	"os"
	"strings"

	"github.com/gdatasoftwareag/tftp/pkg/logging"
	"github.com/gdatasoftwareag/tftp/pkg/tftp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	rootCmd *cobra.Command

	logger logging.Logger
	cfg    config

	logLevel        string
	configFilePath  string
	developmentLogs bool
)

type config struct {
	FSHandlerBaseDir string

	TFTP tftp.Config
}

func main() {
	rootCmd = &cobra.Command{
		Use:   "tftp",
		Short: "Read only tftp server",
	}

	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "logging level to use")
	rootCmd.PersistentFlags().BoolVarP(&developmentLogs, "development-logs", "d", false, "Enable development mode logs")
	rootCmd.PersistentFlags().StringVarP(&configFilePath, "config", "c", "", "path to the config file to load")

	rootCmd.PersistentPreRunE = rootCmdPersistentPreRunE

	rootCmd.AddCommand(versionCmd, serveCmd)

	if err := rootCmd.Execute(); err != nil {
		panic(err.Error())
	}
}

func rootCmdPersistentPreRunE(cmd *cobra.Command, args []string) (err error) {
	if cmd.Name() == versionCmd.Name() {
		return
	}

	if cfg, err = parseConfig(); err != nil {
		return
	}

	logging.ConfigureLogging(
		logging.ParseLevel(logLevel),
		developmentLogs,
		map[string]interface{}{
			"cmd":  cmd.Name(),
			"args": args,
		},
	)

	logger = logging.MustGetLogger()

	logger.Info(
		"Initiating TFTP server",
		zap.String("git_tag", tftp.GitTag),
		zap.String("git_commit", tftp.GitCommit),
		zap.String("build_time", tftp.BuildTime),
	)

	return
}

func parseConfig() (cfg config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if configFilePath == "" {
		if dir, err := os.Getwd(); err == nil {
			viper.AddConfigPath(dir)
		}
		viper.AddConfigPath("/etc/tftp")
	} else {
		viper.SetConfigFile(configFilePath)
	}

	viper.SetDefault("FSHandlerBaseDir", "/var/lib/tftpboot")

	viper.SetDefault("tftp.ip", "0.0.0.0")
	viper.SetDefault("tftp.port", 69)
	viper.SetDefault("tftp.retransmissions", 3)
	viper.SetDefault("tftp.maxparallelconnections", 10)
	viper.SetDefault("tftp.filetransfertimeout", "10s")

	viper.SetDefault("tftp.metrics.enabled", true)
	viper.SetDefault("tftp.metrics.port", 9100)

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("TFTP")
	viper.AutomaticEnv()
	if err = viper.ReadInConfig(); err != nil {
		return
	}
	if err = viper.Unmarshal(&cfg); err != nil {
		return
	}
	return
}
