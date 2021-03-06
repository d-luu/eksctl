// +build integration

package integration_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/weaveworks/eksctl/integration/runner"
	. "github.com/weaveworks/eksctl/integration/runner"
	"github.com/weaveworks/eksctl/pkg/utils/names"
)

const (
	goBackVersions = 2
)

var _ = Describe("(Integration) [Backwards compatibility test]", func() {

	var (
		clusterName   = names.ForCluster("", "")
		initialNgName = "ng-1"
		newNgName     = "ng-2"
	)

	It("should support clusters created with a previous version of eksctl", func() {
		By("downloading a previous release")
		eksctlDir, err := ioutil.TempDir(os.TempDir(), "eksctl")
		Expect(err).ToNot(HaveOccurred())

		defer func() {
			Expect(os.RemoveAll(eksctlDir)).ToNot(HaveOccurred())
		}()

		downloadRelease(eksctlDir)

		eksctlPath := path.Join(eksctlDir, "eksctl")

		version, err := getVersion(eksctlPath)
		Expect(err).ToNot(HaveOccurred())

		By(fmt.Sprintf("creating a cluster with release %q", version))
		cmd := runner.NewCmd(eksctlPath).
			WithArgs(
				"create",
				"cluster",
				"--name", clusterName,
				"--nodegroup-name", initialNgName,
				"-v4",
				"--region", region,
			).
			WithTimeout(20 * time.Minute)

		Expect(cmd).To(RunSuccessfully())

		By("fetching the new cluster")
		cmd = eksctlGetCmd.WithArgs(
			"cluster",
			clusterName,
			"--output", "json",
		)

		Expect(cmd).To(RunSuccessfullyWithOutputString(ContainSubstring(clusterName)))

		By("adding a nodegroup")
		cmd = eksctlCreateCmd.WithArgs(
			"nodegroup",
			"--cluster", clusterName,
			"--nodes", "2",
			newNgName,
		)
		Expect(cmd).To(RunSuccessfully())

		By("scaling the initial nodegroup")
		cmd = eksctlScaleNodeGroupCmd.WithArgs(
			"--cluster", clusterName,
			"--nodes", "3",
			"--name", initialNgName,
		)
		Expect(cmd).To(RunSuccessfully())

		By("deleting the new nodegroup")
		cmd = eksctlDeleteCmd.WithArgs(
			"nodegroup",
			"--verbose", "4",
			"--cluster", clusterName,
			newNgName,
		)
		Expect(cmd).To(RunSuccessfully())

		By("deleting the initial nodegroup")
		cmd = eksctlDeleteCmd.WithArgs(
			"nodegroup",
			"--verbose", "4",
			"--cluster", clusterName,
			initialNgName,
		)
		Expect(cmd).To(RunSuccessfully())

		By("deleting the cluster")
		cmd = eksctlDeleteClusterCmd.WithArgs(
			"--name", clusterName,
		)
		Expect(cmd).To(RunSuccessfully())
	})
})

func downloadRelease(dir string) {
	cmd := runner.NewCmd("./scripts/download-previous-release.sh").
		WithEnv(
			fmt.Sprintf("GO_BACK_VERSIONS=%d", goBackVersions),
			fmt.Sprintf("DOWNLOAD_DIR=%s", dir),
		).
		WithTimeout(30 * time.Second)

	ExpectWithOffset(1, cmd).To(RunSuccessfully())
}

func getVersion(eksctlPath string) (string, error) {
	cmd := runner.NewCmd(eksctlPath).WithArgs("version")
	session := cmd.Run()
	if session.ExitCode() != 0 {
		return "", errors.New(string(session.Err.Contents()))
	}
	return string(session.Buffer().Contents()), nil
}
