package app

import (
	"bufio"
	"fmt"
	"inmemoryStorageString/config"
	"inmemoryStorageString/internal/app/deleter"
	"inmemoryStorageString/internal/app/dumper"
	"inmemoryStorageString/internal/app/storage"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Application struct {
	cfg      *config.Config
	listener net.Listener
	storage  *storage.Service
	logger   *log.Logger
	deleters []*deleter.Service
	dumper   *dumper.Service
}

func New(cfg *config.Config) (*Application, error) {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		log.New(os.Stdout, "Storage ", 0).Fatal(err)
		return nil, err
	}

	items := make(map[string]storage.Item)
	expirations := make(map[int64][]string)
	file, err := os.OpenFile(cfg.DumpFile, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	var deleterChans []chan time.Duration
	var deleterStops []chan bool
	var n = 5
	for n > 0 {
		deleterChan := make(chan time.Duration, 2)
		deleterChans = append(deleterChans, deleterChan)
		deleterStop := make(chan bool)
		deleterStops = append(deleterStops, deleterStop)
		n--
	}
	curStorage := storage.New(cfg, items, expirations, file, deleterChans)
	n = 5
	var deleters []*deleter.Service
	for n > 0 {
		deleter := deleter.New(deleterStops[n-1], curStorage)
		deleters = append(deleters, deleter)
		n--
	}
	stopDump := make(chan bool)
	dumper := dumper.New(time.Duration(cfg.DumpInterval)*time.Second, stopDump, curStorage)

	return &Application{
		cfg:      cfg,
		listener: listener,
		logger:   log.New(os.Stdout, "Storage ", 0),
		storage:  curStorage,
		dumper:   dumper,
	}, nil
}

func (a *Application) Run() error {
	err := a.storage.Load()
	if err != nil {
		a.logger.Println(err)
		return err
	}
	conn, err := a.listener.Accept()
	if err != nil {
		a.logger.Println(err)
		return err
	}

	for _, deleter := range a.deleters {
		go deleter.Run()
	}

	go a.dumper.Run()
	go func() {
		sc := make(chan os.Signal, 1)

		signal.Notify(sc,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT)
		for {
			<-sc
			a.Quit()
			return
		}
	}()
	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		fmt.Println(message)
		if err != nil {
			a.logger.Println(err)
			return err
		}
		var result []byte
		cmd := strings.Split(string(strings.Split(string(message), "\n")[0]), " ")
		if len(cmd) == 2 {
			switch cmd[0] {
			case "get":
				var found bool
				result1, found := a.storage.Get(cmd[1])
				result = []byte(result1)
				if !found {
					result = []byte("not found")
				}
			case "delete":
				result = []byte("delete")
				err := a.storage.Delete(cmd[1])
				if err != nil {
					result = []byte("didn't delete")
				}
			}
		}
		if len(cmd) == 3 {
			result = []byte("put")
			a.storage.Set(cmd[1], cmd[2], 0)
		}
		if len(cmd) == 4 {
			result = []byte("put")
			duration, err := strconv.Atoi(cmd[3])
			if err != nil {
				result = []byte("didn't put")
			}
			a.storage.Set(cmd[1], cmd[2], uint(duration))
		}
		result = append(result, []byte("\n")...)
		_, err = conn.Write(result)
		if err != nil {
			a.logger.Println(err)
			return err
		}
	}
}

func (a *Application) Quit() {
	fmt.Println("Application quit")
	for _, deleter := range a.deleters {
		deleter.Stop()
	}
	a.dumper.Stop()
	a.listener.Close()
	a.storage.Stop()
	os.Exit(0)
}
