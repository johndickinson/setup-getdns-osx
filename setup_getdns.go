package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Key struct {
	packagename, parameter string
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// dwnloadball downloads a file from the url and un-compresses it and extracts
// the archive to disk.
// path: top build dir.
// url: url to download
// name: name of the tarball
func downloadball(path string, url string, name string) (err error) {
	fmt.Println("Downloading: " + url)

	ball, err := http.Get(url)
	if err != nil {
		return err
	}
	defer ball.Body.Close()

	// Decompression
	var archive io.ReadCloser
	var reader *tar.Reader
	if filepath.Ext(name) == ".gz" || filepath.Ext(name) == ".tgz" {
		archive, err = gzip.NewReader(ball.Body)
		if err != nil {
			return err
		}
		defer archive.Close()
		reader = tar.NewReader(archive)
	} else {
		reader = tar.NewReader(ball.Body)
	}
	// De-archiving
	if filepath.Ext(name) == ".zip" {
		// we have a zip file
		file := filepath.Join(path, name)
		out, err := os.Create(file)
		if err != nil {
			return err
		}
		defer out.Close()
		// Write the body to file
		_, err = io.Copy(out, ball.Body)
		if err != nil {
			return err
		}
		Unzip(file, path)
		return nil
	} else {
		// we have a tar file
		for {
			header, err := reader.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			// fmt.Println(header.Name)
			filename := filepath.Join(path, header.Name)
			info := header.FileInfo()
			// fmt.Println(filename)
			if info.IsDir() {
				if err = os.MkdirAll(filename, info.Mode()); err != nil {
					return err
				}
				continue
			}
			file, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
			if err != nil {
				return err
			}
			defer file.Close()
			_, err = io.Copy(file, reader)
			if err != nil {
				return err
			}
		}
	}
	fmt.Println("Done.")
	return nil
}

func main() {
	builddir := "/tmp/"
	installdir := "/Users/jad/setup-getdns-osx-test"
	var stdoutStderr []byte
	var cmd *exec.Cmd
	var srcdir string
	// packagelist := [...]string{"getdns"}
	packagelist := [...]string{"autoconf", "automake", "libtool", "pkgc", "check", "openssl", "libevent", "unbound", "libidn", "getdns"}
	packages := make(map[Key]string)

	packages[Key{"autoconf", "VERSION"}] = "2.69"
	packages[Key{"autoconf", "DIR"}] = "autoconf-" + packages[Key{"autoconf", "VERSION"}]
	packages[Key{"autoconf", "TAR"}] = packages[Key{"autoconf", "DIR"}] + ".tar.gz"
	packages[Key{"autoconf", "URL"}] = "http://ftpmirror.gnu.org/autoconf/" + packages[Key{"autoconf", "TAR"}]
	packages[Key{"autoconf", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"autoconf", "MAKE_CHECK"}] = ""
	packages[Key{"autoconf", "SIG"}] = ""

	packages[Key{"automake", "VERSION"}] = "1.15"
	packages[Key{"automake", "DIR"}] = "automake-" + packages[Key{"automake", "VERSION"}]
	packages[Key{"automake", "TAR"}] = packages[Key{"automake", "DIR"}] + ".tar.gz"
	packages[Key{"automake", "URL"}] = "http://ftpmirror.gnu.org/automake/" + packages[Key{"automake", "TAR"}]
	packages[Key{"automake", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"automake", "MAKE_CHECK"}] = ""
	packages[Key{"automake", "SIG"}] = ""

	packages[Key{"libtool", "VERSION"}] = "2.4.6"
	packages[Key{"libtool", "DIR"}] = "libtool-" + packages[Key{"libtool", "VERSION"}]
	packages[Key{"libtool", "TAR"}] = packages[Key{"libtool", "DIR"}] + ".tar.gz"
	packages[Key{"libtool", "URL"}] = "http://ftpmirror.gnu.org/libtool/" + packages[Key{"libtool", "TAR"}]
	packages[Key{"libtool", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"libtool", "MAKE_CHECK"}] = ""
	packages[Key{"libtool", "SIG"}] = ""

	packages[Key{"pkgc", "VERSION"}] = "0.29.2"
	packages[Key{"pkgc", "DIR"}] = "pkg-config-" + packages[Key{"pkgc", "VERSION"}]
	packages[Key{"pkgc", "TAR"}] = packages[Key{"pkgc", "DIR"}] + ".tar.gz"
	packages[Key{"pkgc", "URL"}] = "https://pkg-config.freedesktop.org/releases/" + packages[Key{"pkgc", "TAR"}]
	packages[Key{"pkgc", "CONFIGURE"}] = "./configure --with-internal-glib --prefix=" + installdir
	packages[Key{"pkgc", "MAKE_CHECK"}] = ""
	packages[Key{"pkgc", "SIG"}] = ""

	packages[Key{"check", "VERSION"}] = "0.11.0"
	packages[Key{"check", "DIR"}] = packages[Key{"check", "VERSION"}]
	packages[Key{"check", "TAR"}] = packages[Key{"check", "DIR"}] + ".zip"
	packages[Key{"check", "URL"}] = "https://github.com/libcheck/check/archive/" + packages[Key{"check", "TAR"}]
	packages[Key{"check", "CONFIGURE"}] = "autoreconf -i && ./configure --prefix=" + installdir
	packages[Key{"check", "MAKE_CHECK"}] = ""
	packages[Key{"check", "SIG"}] = ""

	packages[Key{"libevent", "VERSION"}] = "2.1.8-stable"
	packages[Key{"libevent", "DIR"}] = "libevent-" + packages[Key{"libevent", "VERSION"}]
	packages[Key{"libevent", "TAR"}] = packages[Key{"libevent", "DIR"}] + ".tar.gz"
	packages[Key{"libevent", "URL_PATH"}] = "release-" + packages[Key{"libevent", "VERSION"}]
	packages[Key{"libevent", "URL"}] = "https://github.com/libevent/libevent/releases/download/" + packages[Key{"libevent", "URL_PATH"}] + "/" + packages[Key{"libevent", "TAR"}]
	packages[Key{"libevent", "CONFIGURE"}] = "./configure --prefix=" + installdir + " CPPFLAGS=-I" + installdir + "/include LDFLAGS=\"-Wl,-headerpad_max_install_names -L" + installdir + "/lib\""
	packages[Key{"libevent", "TEST"}] = "make check"
	packages[Key{"libevent", "SIG"}] = ""

	packages[Key{"openssl", "VERSION"}] = "1.1.0f"
	packages[Key{"openssl", "DIR"}] = "openssl-" + packages[Key{"openssl", "VERSION"}]
	packages[Key{"openssl", "TAR"}] = packages[Key{"openssl", "DIR"}] + ".tar.gz"
	packages[Key{"openssl", "URL"}] = "http://openssl.org/source/" + packages[Key{"openssl", "TAR"}]
	packages[Key{"openssl", "CONFIGURE"}] = "./Configure darwin64-x86_64-cc --shared --prefix=" + installdir + " -Wl,-headerpad_max_install_names"
	packages[Key{"openssl", "MAKE_CHECK"}] = "make test"
	packages[Key{"openssl", "SIG"}] = "9e3e02bc8b4965477a7a1d33be1249299a9deb15"

	packages[Key{"unbound", "VERSION"}] = "1.6.3"
	packages[Key{"unbound", "DIR"}] = "unbound-" + packages[Key{"unbound", "VERSION"}]
	packages[Key{"unbound", "TAR"}] = packages[Key{"unbound", "DIR"}] + ".tar.gz"
	packages[Key{"unbound", "URL"}] = "http://unbound.nlnetlabs.nl/downloads/" + packages[Key{"unbound", "TAR"}]
	packages[Key{"unbound", "CONFIGURE"}] = "./configure --prefix=" + installdir + " --with-ssl=" + installdir + " --with-conf-file=. LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"unbound", "MAKE_CHECK"}] = "make test"
	packages[Key{"unbound", "SIG"}] = "4477627c31e8728058565f3bae3a12a1544d8a9c"

	packages[Key{"libidn", "VERSION"}] = "1.33"
	packages[Key{"libidn", "DIR"}] = "libidn-" + packages[Key{"libidn", "VERSION"}]
	packages[Key{"libidn", "TAR"}] = packages[Key{"libidn", "DIR"}] + ".tar.gz"
	packages[Key{"libidn", "URL"}] = "http://ftp.gnu.org/gnu/libidn/" + packages[Key{"libidn", "TAR"}]
	packages[Key{"libidn", "CONFIGURE"}] = "./configure --prefix=" + installdir + " LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"libidn", "MAKE_CHECK"}] = "make check"
	packages[Key{"libidn", "SIG"}] = ""

	packages[Key{"getdns", "VERSION"}] = "1.1.1"
	packages[Key{"getdns", "DIR"}] = "getdns-" + packages[Key{"getdns", "VERSION"}]
	packages[Key{"getdns", "URL_PATH"}] = strings.Replace(packages[Key{"getdns", "DIR"}], ".", "-", -1)
	packages[Key{"getdns", "TAR"}] = packages[Key{"getdns", "DIR"}] + ".tar.gz"
	packages[Key{"getdns", "URL"}] = "https://getdnsapi.net/releases/" + packages[Key{"getdns", "URL_PATH"}] + "/" + packages[Key{"getdns", "TAR"}]
	packages[Key{"getdns", "CONFIGURE"}] = "./configure --prefix=" + installdir + " --with-ssl=" + installdir + " --with-libunbound=" + installdir + " --with-libidn=" + installdir + " --with-libevent --enable-debug-daemon LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"getdns", "MAKE_CHECK"}] = "make check"
	packages[Key{"getdns", "SIG"}] = ""

	os.RemoveAll(installdir)
	for _, v := range packagelist {
		fmt.Println("Cleaning: " + v)
		err := os.RemoveAll(filepath.Join(builddir, packages[Key{v, "DIR"}]))
		if err != nil {
			os.Exit(1)
		}
	}

	os.Setenv("PATH", installdir+"/bin:"+installdir+"/sbin:"+os.Getenv("PATH"))
	env := os.Environ()
	for _, v := range packagelist {
		if v == "check" {
			srcdir = builddir + "check-" + packages[Key{v, "DIR"}]
		} else {
			srcdir = builddir + packages[Key{v, "DIR"}]
		}
		fmt.Println("Building: " + v)

		err := downloadball(builddir, packages[Key{v, "URL"}], packages[Key{v, "TAR"}])
		if err != nil {
			fmt.Println("Download and extract failed " + err.Error())
			os.Exit(1)
		}
		fmt.Println("Configuring: " + v)
		// fmt.Println(packages[Key{v, "CONFIGURE"}])
		cmd = exec.Command("sh", "-c", packages[Key{v, "CONFIGURE"}])
		cmd.Dir = srcdir
		cmd.Env = env
		// fmt.Println(cmd.Env)
		// fmt.Println("Working directory is: " + cmd.Dir)
		stdoutStderr, err = cmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Println("Making: " + v)
		cmd = exec.Command("sh", "-c", "make")
		cmd.Dir = srcdir
		cmd.Env = env
		stdoutStderr, err = cmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		fmt.Println("Installing: " + v)
		cmd = exec.Command("sh", "-c", "make install")
		cmd.Dir = srcdir
		cmd.Env = env
		stdoutStderr, err = cmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

	}

}
