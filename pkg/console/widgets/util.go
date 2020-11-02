package widgets

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/ghodss/yaml"
	"github.com/jroimartin/gocui"
	cfg "github.com/rancher/harvester/pkg/console/config"
	"github.com/rancher/k3os/pkg/config"
)

func ArrowUp(g *gocui.Gui, v *gocui.View) error {
	if v == nil || isAtTop(v) {
		return nil
	}

	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy-1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy-1); err != nil {
			return err
		}
	}
	return nil
}

func ArrowDown(g *gocui.Gui, v *gocui.View) error {
	if v == nil || isAtEnd(v) {
		return nil
	}
	cx, cy := v.Cursor()
	if err := v.SetCursor(cx, cy+1); err != nil {
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+1); err != nil {
			return err
		}
	}

	return nil
}

func isAtTop(v *gocui.View) bool {
	_, cy := v.Cursor()
	if cy == 0 {
		return true
	}
	return false
}

func isAtEnd(v *gocui.View) bool {
	_, cy := v.Cursor()
	lines := len(v.BufferLines())
	if lines < 2 || cy == lines-2 {
		return true
	}
	return false
}

func DoInstall(g *gocui.Gui) error {
	var (
		err      error
		tempFile *os.File
	)
	if cfg.Config.K3OS.Install.ConfigURL == "" {
		tempFile, err = ioutil.TempFile("/tmp", "k3os.XXXXXXXX")
		if err != nil {
			return err
		}
		defer tempFile.Close()

		cfg.Config.K3OS.Install.ConfigURL = tempFile.Name()
	}
	ev, err := config.ToEnv(cfg.Config)
	if err != nil {
		return err
	}
	if tempFile != nil {
		cfg.Config.K3OS.Install = nil
		bytes, err := yaml.Marshal(&cfg.Config)
		if err != nil {
			return err
		}
		if _, err := tempFile.Write(bytes); err != nil {
			return err
		}
		if err := tempFile.Close(); err != nil {
			return err
		}
		defer os.Remove(tempFile.Name())
	}
	cmd := exec.Command("/usr/libexec/k3os/install")
	cmd.Env = append(os.Environ(), ev...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("install")
			if err != nil {
				return err
			}
			fmt.Fprintln(v, m)

			lines := len(v.BufferLines())
			_, sy := v.Size()
			if lines > sy {
				ox, oy := v.Origin()
				v.SetOrigin(ox, oy+1)
			}
			return nil
		})
	}
	scanner = bufio.NewScanner(stderr)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		m := scanner.Text()
		g.Update(func(g *gocui.Gui) error {
			v, err := g.View("install")
			if err != nil {
				return err
			}
			fmt.Fprintln(v, m)

			lines := len(v.BufferLines())
			_, sy := v.Size()
			if lines > sy {
				ox, oy := v.Origin()
				v.SetOrigin(ox, oy+1)
			}
			return nil
		})
	}
	return nil
}
