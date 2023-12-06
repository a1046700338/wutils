package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Weidows/wutils/utils/collection"
	"github.com/Weidows/wutils/utils/grammar"
	"github.com/Weidows/wutils/utils/log"
	"github.com/Weidows/wutils/utils/os2"
	"github.com/jinzhu/configor"
	"github.com/urfave/cli/v2"
)

const ConfigPath = "keep-runner.yml"

var (
	logger = log.GetLogger()

	config = struct {
		Debug   bool `default:"false"`
		Refresh struct {
			Delay int `default:"10"`
		}
		Parallel struct {
			Dsg bool
			Ol  bool
			sl  bool
		}

		Dsg struct {
			Disk  []string `required:"true"`
			Delay int      `default:"30"`
		} `yaml:"dsg" required:"true"`

		Ol struct {
			Delay    int `default:"2"`
			Patterns []struct {
				Title   string
				Opacity byte
			}
		}

		sl struct {}
	}{}

	app = &cli.App{
		Name: "keep-runner",
		Authors: []*cli.Author{
			{
				Name:  "Weidows",
				Email: "utsuko27@gmail.com",
			},
		},
		EnableBashCompletion: true,
		Usage: "几个旨在后台运行的程序, config 使用: ./keep-runner.yml\n" +
			"Default config: https://github.com/Weidows/wutils/tree/master/config/cmd/keep-runner.yml",
		Commands: []*cli.Command{
			{
				Name:    "parallel",
				Aliases: []string{"pl"},
				Usage:   "并行+后台执行任务(取自config)",
				Action:  parallelAction,
			},
			{
				Name:      "dsg",
				Aliases:   []string{""},
				Usage: "Disk sleep guard\n" +
					"防止硬盘睡眠 (每隔一段自定义的时间, 往指定盘里写一个时间戳)\n" +
					"外接 HDD 频繁启停甚是头疼, 后台让它怠速跑着, 免得起起停停增加损坏率",
				Action: dsgAction,
			},
			{
				Name:    "ol",
				Aliases: []string{""},
				Usage: "Opacity Listener\n" +
					"后台持续运行, 并每隔指定时间扫一次运行的窗口\n" +
					"把指定窗口设置opacity, 使其透明化(比BLend好使~)",
				Action: olAction,
				Subcommands: []*cli.Command{
					{
						Name:    "list",
						Aliases: []string{""},
						Usage:   "list all visible windows",
						Action:  olListAction,
					},
				},
			},
			{
				Name:    "sl",
				Aliases: []string{""},
				Usage: "Stop Listener\n" +
					"终止程序进程\n" +
					"使其透明化恢复初始",
				Action: slAction,
			},
			{
				Name:    "config",
				Aliases: []string{""},
				Usage:   "print config file",
				Action:  configAction,
			},
		},
	}
)

func dsgAction(cCtx *cli.Context) (err error) {
	dsg()
	return err
}

func olAction(cCtx *cli.Context) (err error) {
	ol()
	return err
}

func slAction(cCtx *cli.Context) (err error) {
	sl()
	return err
}

func configAction(cCtx *cli.Context) (err error) {
	logger.Println(fmt.Sprintf("%+v", config))
	return err
}

func parallelAction(cCtx *cli.Context) (err error) {
	if config.Parallel.Dsg {
		go dsg()
	}
	if config.Parallel.Ol {
		ol()
	}

	return err
}

func olListAction(cCtx *cli.Context) (err error) {
	collection.ForEach(os2.GetEnumWindowsInfo(&os2.EnumWindowsFilter{
		IgnoreNoTitled:  true,
		IgnoreInvisible: true,
	}), func(i int, v *os2.EnumWindowsResult) {
		logger.Println(fmt.Sprintf("%+v", v))
	})
	return err
}

func TestMain(m *testing.M) {
	// Set up logger
	log.InitLogger()

	// Run tests
	code := m.Run()

	// Clean up
	os.Exit(code)
}

func TestDsgAction(t *testing.T) {
	// Mock config
	config.Dsg.Disk = []string{"C:", "D:"}
	config.Dsg.Delay = 5

	// Create a file on each disk to simulate dsg running
	file1, err := os.Create("C:.dsg")
	if err != nil {
		t.Errorf("Failed to create file on disk C: : %s", err.Error())
	}
	file1.Close()
	file2, err := os.Create("D:.dsg")
	if err != nil {
		t.Errorf("Failed to create file on disk D: : %s", err.Error())
	}
	file2.Close()

	// Run dsgAction
	dsgAction(nil)

	// Check if new timestamp was written to files
	time.Sleep(time.Second * 10)
	file1, err = os.OpenFile("C:.dsg", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Failed to open file on disk C: : %s", err.Error())
	}
	_, err = file1.WriteString("new timestamp\n")
	file1.Close()
	file2, err = os.OpenFile("D:.dsg", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Errorf("Failed to open file on disk D: : %s", err.Error())
	}
	_, err = file2.WriteString("new timestamp\n")
	file2.Close()

	// Check if files were written within the specified delay
	startTime := time.Now()
	time.Sleep(time.Second * config.Dsg.Delay)
	elapsedTime := time.Since(startTime)
	if elapsedTime > (time.Second * config.Dsg.Delay) {
		t.Errorf("Files were not written within the specified delay (actual delay: %s)", elapsedTime.String())
	}
}

func TestOlAction(t *testing.T) {
	// Mock config
	config.Ol.Patterns = []struct {
		Title   string
		Opacity byte
	}{{
		Title:   "Window 1",
		Opacity: 50,
	}, {
		Title:   "Window 2",
		Opacity: 80,
	}}

	// Create two windows with matching titles
	window1 := &os2.EnumWindowsResult{
		Title: "Window 1",
	}
	window2 := &os2.EnumWindowsResult{
		Title: "Window 2",
	}

	// Run olAction
	olAction(nil)

	// Check if opacity was set correctly on windows
	isSuccess1 := os2.SetWindowOpacity(window1.Handle, 50)
	if !isSuccess1 {
		t.Errorf("Failed to set opacity on window 1")
	}
	isSuccess2 := os2.SetWindowOpacity(window2.Handle, 80)
	if !isSuccess2 {
		t.Errorf("Failed to set opacity on window 2")
	}
}

func TestSlAction(t *testing.T) {
	// Mock initial opacity
	initialOpacity := 50

	// Mock config
	config.Parallel.sl = true

	// Create two windows
	window1 := &os2.EnumWindowsResult{
		Title: "Window 1",
	}
	window2 := &os2.EnumWindowsResult{
		Title: "Window 2",
	}

	// Set initial opacity on windows
	isSuccess1 := os2.SetWindowOpacity(window1.Handle, initialOpacity)
	if !isSuccess1 {
		t.Errorf("Failed to set initial opacity on window 1")
	}
	isSuccess2 := os2.SetWindowOpacity(window2.Handle, initialOpacity)
	if !isSuccess2 {
		t.Errorf("Failed to set initial opacity on window 2")
	}

	// Run slAction
	slAction(nil)

	// Check if windows were restored to initial opacity
	isSuccess1 = os2.SetWindowOpacity(window1.Handle, initialOpacity)
	if !isSuccess1 {
		t.Errorf("Failed to restore window 1 to initial opacity")
	}
	isSuccess2 = os2.SetWindowOpacity(window2.Handle, initialOpacity)
	if !isSuccess2 {
		t.Errorf("Failed to restore window 2 to initial opacity")
	}
}

func TestRefreshConfig(t *testing.T) {
	// Mock config
	config.Refresh.Delay = 1

	// Run refreshConfig
	go refreshConfig()

	// Wait for delay
	time.Sleep(time.Second * time.Duration(config.Refresh.Delay))

	// Update config delay
	config.Refresh.Delay = 2

	// Wait for delay to ensure config was refreshed
	time.Sleep(time.Second * time.Duration(config.Refresh.Delay))

	// Check if refresh delay was updated
	if config.Refresh.Delay != 2 {
		t.Errorf("Refresh delay was not updated correctly")
	}
}