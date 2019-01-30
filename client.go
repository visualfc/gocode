package main

import (
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func do_client() int {
	addr := *g_addr
	if *g_sock == "unix" {
		addr = get_socket_filename()
	}

	// client
	client, err := rpc.Dial(*g_sock, addr)
	if err != nil {
		if *g_sock == "unix" && file_exists(addr) {
			os.Remove(addr)
		}

		err = try_run_server()
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			return 1
		}
		client, err = try_to_connect(*g_sock, addr)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			return 1
		}
	}
	defer client.Close()

	if flag.NArg() > 0 {
		switch flag.Arg(0) {
		case "autocomplete":
			cmd_auto_complete(client)
		case "liteide_typesinfo":
			cmd_types_info(client)
		case "close":
			cmd_close(client)
		case "status":
			cmd_status(client)
		case "drop-cache":
			cmd_drop_cache(client)
		case "set":
			cmd_set(client)
		case "options":
			cmd_options(client)
		default:
			fmt.Printf("unknown argument: %q, try running \"gocode -h\"\n", flag.Arg(0))
			return 1
		}
	}
	return 0
}

func try_run_server() error {
	path := get_executable_filename()
	args := []string{os.Args[0], "-s", "-sock", *g_sock, "-addr", *g_addr}
	cwd, _ := os.Getwd()

	var err error
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	stdout, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	stderr, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}

	procattr := os.ProcAttr{Dir: cwd, Env: os.Environ(), Files: []*os.File{stdin, stdout, stderr}}
	p, err := os.StartProcess(path, args, &procattr)
	if err != nil {
		return err
	}

	return p.Release()
}

func try_to_connect(network, address string) (client *rpc.Client, err error) {
	t := 0
	for {
		client, err = rpc.Dial(network, address)
		if err != nil && t < 1000 {
			time.Sleep(10 * time.Millisecond)
			t += 10
			continue
		}
		break
	}

	return
}

func prepare_file_filename_cursor(filter bool) ([]byte, string, int, string) {
	var file []byte
	var err error

	if *g_input != "" {
		file, err = ioutil.ReadFile(*g_input)
	} else {
		file, err = ioutil.ReadAll(os.Stdin)
	}

	if err != nil {
		panic(err.Error())
	}

	filename := *g_input
	cursor := -1

	offset := ""
	addin := ""
	switch flag.NArg() {
	case 2:
		offset = flag.Arg(1)
	case 3:
		filename = flag.Arg(1) // Override default filename
		offset = flag.Arg(2)
	case 4:
		filename = flag.Arg(1) // Override default filename
		offset = flag.Arg(2)
		addin = flag.Arg(3)
	}

	var skipped int
	if filter {
		file, skipped = filter_out_shebang(file)
	}

	if offset != "" {
		if offset[0] == 'c' || offset[0] == 'C' {
			cursor, _ = strconv.Atoi(offset[1:])
			cursor = char_to_byte_offset(file, cursor)
		} else {
			cursor, _ = strconv.Atoi(offset)
		}
	}

	cursor -= skipped

	if filename != "" && !filepath.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = filepath.Join(cwd, filename)
	}
	return file, filename, cursor, addin
}

//-------------------------------------------------------------------------
// commands
//-------------------------------------------------------------------------

func cmd_status(c *rpc.Client) {
	fmt.Printf("%s\n", client_status(c, 0))
}

func cmd_auto_complete(c *rpc.Client) {
	context := pack_build_context(&build.Default)
	file, filename, cursor, _ := prepare_file_filename_cursor(true)
	f := get_formatter(*g_format)
	f.write_candidates(client_auto_complete(c, file, filename, cursor, context))
}

func write_tyepsinfo(infos []string, num int) {
	if infos == nil {
		return
	}
	for _, c := range infos {
		fmt.Printf("%s\n", c)
	}
}

func cmd_types_info(c *rpc.Client) {
	context := pack_build_context(&build.Default)
	file, filename, cursor, addin := prepare_file_filename_cursor(true)
	write_tyepsinfo(client_types_info(c, file, filename, cursor, addin, context))
}

func cmd_close(c *rpc.Client) {
	client_close(c, 0)
}

func cmd_drop_cache(c *rpc.Client) {
	client_drop_cache(c, 0)
}

func cmd_set(c *rpc.Client) {
	switch flag.NArg() {
	case 1:
		fmt.Print(client_set(c, "\x00", "\x00"))
	case 2:
		fmt.Print(client_set(c, flag.Arg(1), "\x00"))
	case 3:
		fmt.Print(client_set(c, flag.Arg(1), flag.Arg(2)))
	}
}

func cmd_options(c *rpc.Client) {
	fmt.Print(client_options(c, 0))
}
