package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/luke-mayer/application-aggregator/internal/config"
	"github.com/luke-mayer/application-aggregator/internal/database"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	allCommands map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.allCommands[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	cmdFunc, ok := c.allCommands[cmd.name]
	if !ok {
		err := fmt.Sprintf("error in run() - command %s not found in commands", cmd.name)
		return errors.New(err)
	}

	if err := cmdFunc(s, cmd); err != nil {
		newErr := fmt.Sprintf("error in run() - command %s function call returned error: %s", cmd.name, err)
		return errors.New(newErr)
	}
	return nil
}

func handlerLogin(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error in handlerLogin() - login requires a username argument")
	}

	if len(cmd.args) < 1 {
		return errors.New("error in handlerLogin() - a user name is required")
	}

	userName := cmd.args[0]
	s.cfg.SetUser(userName)
	fmt.Printf("User has been set to: %s\n", userName)
	return nil
}

func handlerCreateApplication(s *state, cmd command) error {
	if cmd.args == nil {
		fmt.Println("Usage: add_application <company> <job title> <preference (-1,0,1)>")
		return errors.New("error in handlerCreateApplication() - no arguments")
	}

	if len(cmd.args) < 3 {
		fmt.Println("Usage: add_application <company> <job title> <preference (-1,0,1)>")
		return errors.New("error in handlerCreateApplication() - not enough arguments provided")
	}

	pref, err := strconv.Atoi(cmd.args[2])
	if err != nil {
		fmt.Println("Usage: add_application \"<company>\" \"<job title>\" <preference (-1,0,1)>")
		return errors.New("error in handlerCreateApplication() - preference argument not an integer")
	}

	params := database.CreateApplicationParams{
		Company:     cmd.args[0],
		Job:         cmd.args[1],
		AppliedDate: time.Now(),
		Preference:  int32(pref),
	}

	_, err = s.db.CreateApplication(context.Background(), params)
	if err != nil {
		newErr := fmt.Sprintf("error in handlerCreateApplication() - failed to create application: %s", err)
		return errors.New(newErr)
	}

	fmt.Printf("Application for position \"%s\" at company \"%s\" with preference \"%v\" successfully added!\n", cmd.args[1], cmd.args[0], pref)

	return nil
}

func handlerGetJobs(s *state, cmd command) error {
	if cmd.args == nil {
		return errors.New("error in handlerGetJobs() - no arguments")
	}

	companies_jobs, err := s.db.GetCompaniesJobs(context.Background())
	if err != nil {
		newErr := fmt.Sprintf("error in handlerGetJobs() - failed to retrieve companies and jobs: %s", err)
		return errors.New(newErr)
	}

	fmt.Println("All jobs (and companies) applied to: ")
	for _, job_comp := range companies_jobs {
		fmt.Printf("Company: %s, Job: %s\n", job_comp.Company, job_comp.Job)
	}
	fmt.Println(companies_jobs)

	return nil
}

func main() {
	var s state

	temp_cfg := config.Read() // retrieves state from application-tracker-config.json

	s.cfg = &temp_cfg

	db, err := sql.Open("postgres", s.cfg.DBUrl)
	if err != nil {
		log.Fatalf("fatal error - issue opening connection to database: %s\n", err)
	}
	s.db = database.New(db)

	cmds := commands{
		allCommands: make(map[string]func(*state, command) error),
	}

	cmds.register("login", handlerLogin)
	cmds.register("add_application", handlerCreateApplication)
	cmds.register("get_jobs", handlerGetJobs)

	args := os.Args
	if len(args) < 2 {
		log.Fatalln("fatal error - less than 2 arguments provided")
	}

	cmd := command{
		name: args[1],
		args: args[2:],
	}

	err = cmds.run(&s, cmd) //runs the called command
	if err != nil {
		log.Fatalln(err)
	}
}
