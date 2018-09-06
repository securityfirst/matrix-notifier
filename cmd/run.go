package cmd

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/securityfirst/matrix-notifier/server"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Runs the server",
	Long:  `Runs the server`,
	Run: func(cmd *cobra.Command, args []string) {
		db, err := conf.GetDB()
		if err != nil {
			logger.Fatalln("DB:", err)
		}
		mailer, err := conf.GetMailer()
		if err != nil {
			logger.Fatalln("Mailer:", err)
		}
		matrix, err := conf.GetMatrix()
		if err != nil {
			logger.Fatalln("Matrix:", err)
		}
		s := server.NewServer(conf.Server.Address, db, matrix, mailer, []byte(conf.Server.Secret))
		logger.Println("Listening on:", conf.Server.Address)
		go func() {
			if err := s.Run(); err != nil && err != http.ErrServerClosed {
				logger.Fatalln("Startup:", err)
			}
		}()

		quit := make(chan os.Signal)
		signal.Notify(quit, os.Interrupt)
		<-quit

		ctx, exit := context.WithTimeout(context.Background(), 5*time.Second)
		defer exit()
		if err := s.Shutdown(ctx); err != nil {
			logger.Fatalln("Shutdown:", err)
		}

	},
}

func init() {
	RootCmd.AddCommand(runCmd)
}
