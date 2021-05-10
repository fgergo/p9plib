// p9plib helps rewriting plan9port programs from C to Go.
// It's temporary. ha!
package p9plib

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
)

type Stdio9pserve struct {
	Stdin9pserve io.WriteCloser
	Stdout9pserve io.ReadCloser
}

func (a Stdio9pserve) Read(p []byte) (int, error) {
	return a.Stdout9pserve.Read(p)
}

func (a Stdio9pserve) Write(p []byte) (int, error) {
	return a.Stdin9pserve.Write(p)
}

// https://9fans.github.io/plan9port/man/man3/post9pservice.html
func Post9pservice(srv *Stdio9pserve, name string) error {
	if name == "" {
		return errors.New("post9pservice: nothing to do")
	}

	addr := ""
	if strings.Contains(name, "!") { // assume name is already network address
		addr = name
	} else {
		ns, err := Getns()
		if err != nil {
			return err
		}
		addr = fmt.Sprintf("unix!%s/%s", ns, name)
	}

	var err error
	cmd := exec.Command("9pserve", "-l", "-n", addr)
	srv.Stdin9pserve, err = cmd.StdinPipe()
	if err != nil {
		return err
	}

	srv.Stdout9pserve, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	return nil
}

func nsfromdisplay() (string, error) {
	disp := os.Getenv("DISPLAY")

	if disp == "" && runtime.GOOS == "darwin" {
		disp = ":0.0"
	}
	
	if disp == "" && runtime.GOOS == "windows" {
		return "", errors.New("environment variable NAMESPACE not set")
	}

	if disp == "" {
		return "", errors.New("$DISPLAY not set")
	}

	// canonicalize: xxx:0.0 => xxx:0
	if strings.HasSuffix(disp, ":0") {
		disp = disp[:len(disp)-2]
	}

	/* turn /tmp/launch/:0 into _tmp_launch_:0 (OS X 10.5) */
	disp = strings.ReplaceAll(disp, "/", "_")

	u, err := user.Current()
	if err != nil {
		return "", err
	}
	ns := fmt.Sprintf("/tmp/ns.%s.%s", u.Username, disp)

	err = os.Mkdir(ns, 0700)
	if err == nil {
		return ns, nil
	}

	d, err := os.Stat(ns)
	if err != nil {
		return "", errors.New(fmt.Sprintf("stat %s, %v", ns, err))
	}

	if (d.Mode() & 0777) != 0700 {		// TODO: add check '|| uid!=owner' of namespace directory
		return "", errors.New(fmt.Sprintf("bad name space dir %s", ns))
	}

	return ns, nil
}

// https://9fans.github.io/plan9port/man/man3/getns.html
func Getns() (string, error) {
	ns := os.Getenv("NAMESPACE")
	if ns != "" {
		return ns, nil
	}

	ns, err := nsfromdisplay()
	if err != nil {
		return ns, err
	}
	return "", errors.New("$NAMESPACE not set")
}
