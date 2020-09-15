package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/jchenrev/assh/humanize"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var envsToSearch = []string{"prod", "dev", "stg", "sandbox"}

func runSSH(cmd *cobra.Command, args []string) {
	var err error
	if len(args) == 1 && strings.HasPrefix(args[0], "i-") {
		err = sshByInstanceID(args[0])
	} else if len(args) == 2 {
		err = sshByRole(args[0], args[1])
	} else if len(args) == 3 {
		err = sshByRoleProfile(args[0], args[1], args[2])
	} else {
		err = newInvalidCommandError()
	}

	if err != nil {
		switch err.(type) {
		case *invalidCommandError:
			fmt.Printf("Error: %v\n%v\n", err, err.(*invalidCommandError).Help)
			os.Exit(1)
		default:
			fmt.Println("Error: ", err)
			os.Exit(255)
		}
	}
}

func sshByInstanceID(instanceID string) error {
	filters := getInstanceIDFilters(instanceID)

	for _, env := range envsToSearch {
		client, err := newEC2Client(env)
		if err != nil {
			return err
		}

		result, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: filters,
		})
		if err != nil {
			return err
		}

		if len(result.Reservations) != 0 && len(result.Reservations[0].Instances) != 0 {
			instance := result.Reservations[0].Instances[0]
			privateIP := *instance.PrivateIpAddress
			fmt.Print(getInstanceInfoString(instance))
			systemSSHCmd := exec.Command("ssh", privateIP)
			systemSSHCmd.Stdout = os.Stdout
			systemSSHCmd.Stdin = os.Stdin
			systemSSHCmd.Stderr = os.Stderr
			err = systemSSHCmd.Run()
			if err != nil {
				return err
			}
		}

	}

	return errors.New("no instances found")
}

func sshByRole(env, role string) error {
	return sshByRoleProfile(env, role, "")
}

func sshByRoleProfile(env, role, profile string) error {
	client, err := newEC2Client(env)
	if err != nil {
		return err
	}

	filters := getRoleProfileFilters(role, profile)

	result, err := client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: filters,
	})
	if err != nil {
		return err
	}

	privateIPs, textLines := getPrivateIPsAndTextLines(result)
	if privateIPs == nil {
		return errors.New("no instances found")
	}

	prompt := promptui.Select{
		Label:    "  " + textLines[0],
		Items:    textLines[1:],
		HideHelp: true,
		Searcher: func(input string, index int) bool {
			return strings.Contains(textLines[index+1], input)
		},
		Size: 30,
		StartInSearchMode: true,
	}
	index, _, err := prompt.Run()
	if err != nil {
		return err
	}

	systemSSHCmd := exec.Command("ssh", privateIPs[index])
	systemSSHCmd.Stdout = os.Stdout
	systemSSHCmd.Stdin = os.Stdin
	systemSSHCmd.Stderr = os.Stderr
	err = systemSSHCmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func newEC2Client(env string) (*ec2.EC2, error) {
	sess, err := session.NewSessionWithOptions(session.Options{
		Profile:           env,
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	return ec2.New(sess), nil
}

func getInstanceIDFilters(instanceID string) []*ec2.Filter {
	key := "instance-id"
	return []*ec2.Filter{
		&ec2.Filter{
			Name:   &key,
			Values: []*string{&instanceID},
		},
	}
}

func getRoleProfileFilters(role, profile string) []*ec2.Filter {
	role = "*" + role + "*"
	tagKeyRole := "tag:role"
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name:   &tagKeyRole,
			Values: []*string{&role},
		},
	}
	if profile != "" {
		tagKeyProfile := "tag:profile"
		filters = append(filters, &ec2.Filter{
			Name:   &tagKeyProfile,
			Values: []*string{&profile},
		})
	}
	return filters
}

func getPrivateIPsAndTextLines(result *ec2.DescribeInstancesOutput) ([]string, []string) {
	var privateIPs []string
	buf := bytes.NewBufferString("")
	writer := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	fmt.Fprintln(writer, "Name\tInstanceID\tPrivateIP\tRole\tType\tState\tUptime")
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			privateIP := "N/A"
			if instance.PrivateIpAddress != nil {
				privateIP = *instance.PrivateIpAddress
			}
			fields := []string{
				*getValueFromTags(instance.Tags, "Name"),
				*instance.InstanceId,
				privateIP,
				formatRoleProfileFromTags(instance.Tags),
				*instance.InstanceType,
				*instance.State.Name,
				humanize.Time(*instance.LaunchTime),
			}
			fmt.Fprintln(writer, strings.Join(fields, "\t"))
			privateIPs = append(privateIPs, privateIP)
		}
	}
	writer.Flush()
	availableOptions := strings.Split(buf.String(), "\n")
	return privateIPs, availableOptions
}

func getInstanceInfoString(instance *ec2.Instance) string {
	buffer := bytes.NewBufferString("")
	writer := tabwriter.NewWriter(buffer, 0, 0, 1, ' ', 0)
	fmt.Fprintf(writer, "Name:\t%v\n", *getValueFromTags(instance.Tags, "Name"))
	fmt.Fprintf(writer, "Instance ID:\t%v\n", *instance.InstanceId)
	fmt.Fprintf(writer, "Private IP:\t%v\n", *instance.PrivateIpAddress)
	fmt.Fprintf(writer, "Environment:\t%v\n", *getValueFromTags(instance.Tags, "environment"))
	fmt.Fprintf(writer, "Role:\t%v\n", *getValueFromTags(instance.Tags, "role"))
	fmt.Fprintf(writer, "Profile:\t%v\n", *getValueFromTags(instance.Tags, "profile"))
	fmt.Fprintf(writer, "Type:\t%v\n", *instance.InstanceType)
	fmt.Fprintf(writer, "State:\t%v\n", *instance.State.Name)
	fmt.Fprintf(writer, "Uptime:\t%v\n", humanize.Time(*instance.LaunchTime))
	writer.Flush()
	return buffer.String()
}

func getValueFromTags(tags []*ec2.Tag, key string) *string {
	for _, tag := range tags {
		if *tag.Key == key {
			return tag.Value
		}
	}
	empty := ""
	return &empty
}

func formatRoleProfileFromTags(tags []*ec2.Tag) string {
	role := getValueFromTags(tags, "role")
	profile := getValueFromTags(tags, "profile")
	if profile != nil && *profile != "" {
		return fmt.Sprintf("%s/%s", *role, *profile)
	}
	return *role
}
