/*
Copyright © 2024 Juliano Martinez

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "haproxy-sticky-tables-exporter",
	Short: "HAProxy Sticky Tables Exporter for Prometheus",
	Long: `HAProxy Sticky Tables Exporter collects metrics from HAProxy stick tables
and exposes them via an HTTP(s) server for Prometheus to scrape.`,
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
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringP("haproxy-config", "c", "/etc/haproxy/haproxy.cfg", "Path to HAProxy configuration file")
	rootCmd.PersistentFlags().IntP("port", "p", 9366, "Port to expose metrics on")
	rootCmd.PersistentFlags().StringSlice("metrics", []string{}, "Comma-separated list of metrics to collect (empty means all)")
	rootCmd.PersistentFlags().StringSlice("tables", []string{}, "Comma-separated list of stick tables to collect (empty means all)")
	viper.BindPFlag("haproxy-config", rootCmd.PersistentFlags().Lookup("haproxy-config"))
	viper.BindPFlag("port", rootCmd.PersistentFlags().Lookup("port"))
	viper.BindPFlag("metrics", rootCmd.PersistentFlags().Lookup("metrics"))
	viper.BindPFlag("tables", rootCmd.PersistentFlags().Lookup("tables"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".haproxy-sticky-tables-exporter" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".haproxy-sticky-tables-exporter")
	}
	viper.SetEnvPrefix("HSTE")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
