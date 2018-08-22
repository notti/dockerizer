package main

import (
	"archive/tar"
	"bytes"
	"debug/elf"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"syscall"
)

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

func getInterp(fname string) (string, error) {
	binary, err := elf.Open(fname)
	if err != nil {
		return "", nil
	}
	for _, prog := range binary.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		tmp := make([]byte, prog.Filesz)
		n, err := prog.ReadAt(tmp, 0)
		if err != nil {
			log.Fatalln("Error during determining interp:", err)
		}
		if n != int(prog.Filesz) {
			log.Fatalln("Couldn't read interp fully")
		}
		return string(tmp[:clen(tmp)]), nil
	}
	return "", nil
}

func parseLibs(out []byte) (ret []string) {
	lines := bytes.Split(out, []byte("\n"))
	for _, line := range lines {
		splitline := bytes.SplitN(line, []byte("=>"), 2)
		if len(splitline) != 2 {
			continue
		}
		libline := bytes.TrimSpace(splitline[1])
		lib := string(libline[:bytes.IndexByte(libline, ' ')])
		log.Println("	Found lib:", lib)
		ret = append(ret, lib)
	}
	return ret
}

func getLibsLdSO(interp, fname string) (ret []string, err error) {
	interpOut, err := exec.Command(interp, "--list", fname).Output()
	if err != nil {
		return
	}
	return parseLibs(interpOut), nil
}

func getLibsLdd(fname string) (ret []string, err error) {
	interpOut, err := exec.Command("ldd", fname).Output()
	if err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			if exit.Sys().(syscall.WaitStatus).ExitStatus() == 1 {
				log.Println("	Not a dynamic lib/executable")
				return nil, nil
			}
		}
		return
	}
	return parseLibs(interpOut), nil
}

func makeFinfo(fname, dir string) (ret *tar.Header, err error) {
	finfo, err := os.Stat(fname)
	if err != nil {
		return
	}
	ret, err = tar.FileInfoHeader(finfo, "")
	if err != nil {
		return
	}
	if dir == "" {
		ret.Name = fname
		return
	}
	ret.Name = path.Join(dir, ret.Name)
	return
}

func appendFiles(tw *tar.Writer, files []string, dir string) error {
	sort.Strings(files)
	var last string
	for _, fname := range files {
		if fname == last {
			continue
		}
		last = fname
		fi, err := makeFinfo(fname, dir)
		if err != nil {
			return err
		}
		err = tw.WriteHeader(fi)
		if err != nil {
			return err
		}
		src, err := os.Open(fname)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(tw, src)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	output := flag.String("out", "", "Output tarfile")
	flag.Parse()
	if *output == "" {
		log.Fatalln("Need an output tarfile")
	}
	var binaries, interps, libs []string
	for _, fname := range flag.Args() {
		log.Println("Analyzing", fname)
		binaries = append(binaries, fname)

		interp, err := getInterp(fname)
		if err != nil {
			log.Fatalln("Couldn't determine interp:", err)
		}
		if interp == "" {
			l, err := getLibsLdd(fname)
			if err != nil {
				log.Fatalln("Couldn't get libs:", err)
			}
			libs = append(libs, l...)
			continue
		}

		interps = append(interps, interp)

		log.Println("	Found interpreter:", interp)

		l, err := getLibsLdSO(interp, fname)
		if err != nil {
			log.Fatalln("Couldn't get libraries:", err)
		}
		libs = append(libs, l...)
	}

	if len(binaries) == 0 {
		log.Fatalln("Need at least a binary as argument")
	}

	out, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln("Error opening output file:", err)
	}
	tw := tar.NewWriter(out)
	defer tw.Close()

	err = appendFiles(tw, binaries, "/")
	if err != nil {
		log.Fatalln(err)
	}
	err = appendFiles(tw, interps, "")
	if err != nil {
		log.Fatalln(err)
	}
	err = appendFiles(tw, libs, "/usr/lib")
	if err != nil {
		log.Fatalln(err)
	}

}
