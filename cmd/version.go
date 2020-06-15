package cmd

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Hugo",
	Long:  `All software has versions. This is Hugo's`,
	Run:   runVersion,
}

// func init() {
// 	rootCmd.AddCommand(versionCmd)
// }

func runVersion(cmd *cobra.Command, args []string) {
	buf := bytes.NewBufferString("")
	w := tabwriter.NewWriter(buf, 0, 0, 1, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "a\tb\tc")
	fmt.Fprintln(w, "aa\tbb\tcc")
	fmt.Fprintln(w, "aaaa\tdddd\teeee")
	w.Flush()
	fmt.Println(buf)
}
