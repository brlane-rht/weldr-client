// Copyright 2021 by Red Hat, Inc. All rights reserved.
// Use of this source is goverend by the Apache License
// that can be found in the LICENSE file.

package compose

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/osbuild/weldr-client/v2/cloud"
	"github.com/osbuild/weldr-client/v2/cmd/composer-cli/root"
)

var (
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "List the detailed status of all composes",
		Example: `  composer-cli compose status
  composer-cli compose status --json`,
		RunE: status,
		Args: cobra.NoArgs,
	}
)

func init() {
	composeCmd.AddCommand(statusCmd)
}

func status(cmd *cobra.Command, args []string) (rcErr error) {
	w := tabwriter.NewWriter(os.Stdout, 5, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tStatus\tTime\tBlueprint\tVersion\tType\tSize")

	// Get the weldrapi composes first
	// Response to any errors happens after checking the cloudapi
	// This is so that any weldrapi composes can be filtered out of the cloudapi
	// response.
	weldrComposes, errors, err := root.Client.ListComposes()
	var weldrUUIDs []string
	for _, wc := range weldrComposes {
		weldrUUIDs = append(weldrUUIDs, wc.ID)
	}

	// Check cloudapi for composes first
	if root.Cloud.Exists() {
		composes, _ := root.Cloud.ListComposes()
		// Filter out any weldrapi composes
		composes = cloud.FilterComposes(composes, nil, weldrUUIDs)
		if len(composes) > 0 {
			for _, compose := range composes {
				// Get as much detail as we can about the compose
				// This depends on the type of build and how it was started so some fields may
				// be blank. Currently no details are available so they are left blank.
				bpName, bpVersion, imageType, size := composeDetails(compose.ID)
				fmt.Fprintf(w, "%s\t%-8s\t%s\t%-15s\t%s\t%-16s\t%s\n", compose.ID,
					root.Cloud.StatusMap(compose.Status),
					"",
					bpName, bpVersion, imageType, size)
			}
		}
	}

	// Handle weldr errors after printing cloudapi composes
	if err != nil {
		return root.ExecutionError(cmd, "List Error: %s", err)
	}
	if len(errors) > 0 {
		rcErr = root.ExecutionErrors(cmd, errors)
	}

	for _, c := range weldrComposes {
		// Convert the API's float64 time to Time
		var s float64
		if c.JobFinished > 0 {
			s = c.JobFinished
		} else if c.JobStarted > 0 {
			s = c.JobStarted
		} else if c.JobCreated > 0 {
			s = c.JobCreated
		}
		sec := int64(s)
		nsec := int64(1 / (s - float64(sec)))
		t := time.Unix(sec, nsec)

		var size string
		if c.Size > 0 {
			size = fmt.Sprintf("%d", c.Size)
		}

		fmt.Fprintf(w, "%s\t%-8s\t%s\t%-15s\t%s\t%-16s\t%s\n", c.ID, c.Status, t.Format("Mon Jan 2 15:04:05 2006"),
			c.Blueprint, c.Version, c.Type, size)
	}
	w.Flush() //nolint:errcheck

	return rcErr
}
