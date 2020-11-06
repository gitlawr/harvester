package console

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	cfg "github.com/rancher/harvester/pkg/console/config"
	"github.com/rancher/k3os/pkg/config"
)

func getEncrptedPasswd(pass string) (string, error) {
	oldShadow, err := ioutil.ReadFile("/etc/shadow")
	if err != nil {
		return "", err
	}
	defer func() {
		ioutil.WriteFile("/etc/shadow", oldShadow, 0640)
	}()

	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("rancher:%s", pass))
	errBuffer := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = errBuffer

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(errBuffer.Bytes())
		return "", err
	}
	f, err := os.Open("/etc/shadow")
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) > 1 && fields[0] == "rancher" {
			return fields[1], nil
		}
	}

	return "", scanner.Err()
}

func getSSHKeysFromURL(url string) ([]string, error) {
	client := http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body := strings.TrimSuffix(string(b), "\n")
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil, fmt.Errorf("got unexpected status code: %d, body: %s", resp.StatusCode, body)
	}
	return strings.Split(body, "\n"), nil
}

func showNext(c *Console, title string, names ...string) error {
	if title != "" {
		titleV, err := c.GetElement(titlePanel)
		if err != nil {
			return err
		}
		titleV.SetContent(title)
	}

	showNoteV := false

	for _, name := range names {
		v, err := c.GetElement(name)
		if err != nil {
			return err
		}
		if err := v.Show(); err != nil {
			return err
		}
		if name == notePanel {
			showNoteV = true
		}
	}

	validatorV, err := c.GetElement(validatorPanel)
	if err != nil {
		return err
	}
	validatorV.Close()
	if !showNoteV {
		noteV, err := c.GetElement(notePanel)
		if err != nil {
			return err
		}
		noteV.Close()
	}
	return nil
}

func setNote(c *Console, msg string) error {
	noteV, err := c.GetElement(notePanel)
	if err != nil {
		return err
	}
	noteV.SetContent(msg)
	if _, err := c.Gui.SetViewOnTop(notePanel); err != nil {
		return err
	}
	return noteV.Show()
}

func setValidator(c *Console, msg string) error {
	validatorV, err := c.GetElement(validatorPanel)
	if err != nil {
		return err
	}
	validatorV.SetContent(msg)
	if _, err := c.Gui.SetViewOnTop(validatorPanel); err != nil {
		return err
	}
	return validatorV.Show()
}

func customizeConfig() {
	//common configs for both server and agent
	cfg.Config.K3OS.DNSNameservers = []string{"8.8.8.8"}
	cfg.Config.K3OS.NTPServers = []string{"ntp.ubuntu.com"}
	cfg.Config.K3OS.Modules = []string{"kvm"}

	if installMode == modeJoin && nodeRole == nodeRoleCompute {
		return
	}

	harvesterChartValues["minio.persistence.size"] = "20Gi"
	harvesterChartValues["containers.apiserver.image.imagePullPolicy"] = "IfNotPresent"
	harvesterChartValues["containers.apiserver.image.tag"] = "master-head"
	harvesterChartValues["service.harvester.type"] = "LoadBalancer"
	harvesterChartValues["containers.apiserver.authMode"] = "localUser"
	harvesterChartValues["minio.mode"] = "distributed"

	cfg.Config.WriteFiles = []config.File{
		{
			Owner:              "root",
			Path:               "/var/lib/rancher/k3s/server/manifests/harvester.yaml",
			RawFilePermissions: "0600",
			Content:            getHarvesterManifestContent(harvesterChartValues),
		},
	}
	cfg.Config.K3OS.K3sArgs = []string{
		"server",
		"--disable",
		"local-storage",
		"--flannel-backend",
		"none",
	}
}

func getHarvesterManifestContent(values map[string]string) string {
	base := `apiVersion: v1
kind: Namespace
metadata:
  name: harvester-system
---
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: harvester
  namespace: kube-system
spec:
  chart: https://%{KUBERNETES_API}%/static/charts/harvester-0.1.0.tgz
  targetNamespace: harvester-system
  set:
`
	var buffer = bytes.Buffer{}
	buffer.WriteString(base)
	for k, v := range values {
		buffer.WriteString(fmt.Sprintf("    %s: %q\n", k, v))
	}
	return buffer.String()
}
