package main

import (
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/andrew-d/go-termutil"
	"github.com/codegangsta/cli"
	"github.com/lukasmartinelli/pgclimb/formats"
	"github.com/lukasmartinelli/pgclimb/pg"
)

type ExportParams struct {
	connStr string
	query   string
	writer  io.Writer
}

func changeHelpTemplateArgs(args string) {
	cli.CommandHelpTemplate = strings.Replace(cli.CommandHelpTemplate, "[arguments...]", args, -1)
}

func isTplFile(arg string) bool {
	return strings.HasSuffix(arg, ".tpl")
}

func parseTemplate(arg string) string {
	if isTplFile(arg) {
		filename := arg
		rawTemplate, err := ioutil.ReadFile(filename)
		if err != nil {
			log.Fatalln(err)
		}
		return string(rawTemplate)
	} else {
		return arg
	}
}

func parseWriter(c *cli.Context) io.Writer {
	outputFilename := c.GlobalString("output")

	if outputFilename != "" {
		f, err := os.Create(outputFilename)
		exitOnError(err)
		return f
	}
	return os.Stdout
}

func exportFormat(c *cli.Context, format formats.DataFormat) {
	connStr := pg.ParseConnStr(c)
	query, err := parseQuery(c)
	exitOnError(err)
	err = formats.Export(query, connStr, format)
	exitOnError(err)
}

func parseQuery(c *cli.Context) (string, error) {
	filename := c.GlobalString("file")
	if filename != "" {
		query, err := ioutil.ReadFile(filename)
		return string(query), err
	}

	command := c.GlobalString("command")
	if command != "" {
		return command, nil
	}

	if !termutil.Isatty(os.Stdin.Fd()) {
		query, err := ioutil.ReadAll(os.Stdin)
		return string(query), err
	}

	return "", errors.New("You need to specify a SQL query.")
}

func exitOnError(err error) {
	log.SetFlags(0)
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "pgclimb"
	app.Version = "0.1"
	app.Usage = "Export data from PostgreSQL into different data formats"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "dbname, d",
			Value:  "postgres",
			Usage:  "database",
			EnvVar: "DB_NAME",
		},
		cli.StringFlag{
			Name:   "host",
			Value:  "localhost",
			Usage:  "host name",
			EnvVar: "DB_HOST",
		},
		cli.StringFlag{
			Name:   "port, p",
			Value:  "5432",
			Usage:  "port",
			EnvVar: "DB_PORT",
		},
		cli.StringFlag{
			Name:   "username, U",
			Value:  "postgres",
			Usage:  "username",
			EnvVar: "DB_USER",
		},
		cli.BoolFlag{
			Name:  "ssl",
			Usage: "require ssl mode",
		},
		cli.StringFlag{
			Name:   "password, pass",
			Value:  "",
			Usage:  "password",
			EnvVar: "DB_PASS",
		},
		cli.StringFlag{
			Name:   "query, command, c",
			Value:  "",
			Usage:  "SQL query to execute",
			EnvVar: "DB_QUERY",
		},
		cli.StringFlag{
			Name:  "file, f",
			Value: "",
			Usage: "SQL query filename",
		},
		cli.StringFlag{
			Name:  "output, o",
			Value: "",
			Usage: "Output filename",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "template",
			Usage: "Export data with custom template",
			Action: func(c *cli.Context) {
				changeHelpTemplateArgs("<template>")

				templateArg := c.Args().First()
				if templateArg == "" {
					cli.ShowCommandHelp(c, "template")
					os.Exit(1)
				}

				rawTemplate := parseTemplate(templateArg)
				writer := parseWriter(c)
				exportFormat(c, formats.NewTemplateFormat(writer, rawTemplate))
			},
		},
		{
			Name:  "jsonlines",
			Usage: "Export newline-delimited JSON objects",
			Action: func(c *cli.Context) {
				format := formats.NewJSONLinesFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "json",
			Usage: "Export JSON document",
			Action: func(c *cli.Context) {
				format := formats.NewJSONArrayFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "csv",
			Usage: "Export CSV",
			Action: func(c *cli.Context) {
				format := formats.NewCsvFormat(parseWriter(c), ';')
				exportFormat(c, format)
			},
		},
		{
			Name:  "tsv",
			Usage: "Export TSV",
			Action: func(c *cli.Context) {
				format := formats.NewCsvFormat(parseWriter(c), '\t')
				exportFormat(c, format)
			},
		},
		{
			Name:  "xml",
			Usage: "Export XML",
			Action: func(c *cli.Context) {
				format := formats.NewXMLFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
		{
			Name:  "xlsx",
			Usage: "Export XLSX spreadsheets",
			Action: func(c *cli.Context) {
				format := formats.NewXlsxFormat(parseWriter(c))
				exportFormat(c, format)
			},
		},
	}

	app.Run(os.Args)
}
