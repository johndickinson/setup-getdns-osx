package main

import (
	"archive/tar"
	"archive/zip"
	// "bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Key struct {
	packagename, parameter string
}

func Writecerts(file string) {
	var cmd *exec.Cmd
	var validcerts []string
	cmd = exec.Command("sh", "-c", "/usr/bin/security find-certificate -a -p /System/Library/Keychains/SystemRootCertificates.keychain")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	re := regexp.MustCompile("(?sm)^(-----BEGIN CERTIFICATE-----.*?-----END CERTIFICATE-----)$")
	result := re.FindAllStringSubmatch(string(stdoutStderr), -1)
	for i := range result {
		// Output in result[i][1]
		cmd = exec.Command("sh", "-c", "openssl x509 -inform pem -checkend 0 -noout")
		stdin, err := cmd.StdinPipe()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, result[i][1])
		}()
		err = cmd.Start()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = cmd.Wait()
		if err != nil {
			continue
		}
		validcerts = append(validcerts, result[i][1]+"\n")
	}

	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer f.Close()
	for i := range validcerts {
		f.WriteString(validcerts[i])
	}
}

func Runcmd(c string, srcdir string, installdir string, path string, ucc bool) int {
	fmt.Println(c)
	if ucc {
		os.Setenv("PATH", "/usr/local/opt/ccache/libexec:"+installdir+"/bin:"+installdir+"/sbin:"+path)
	} else {
		os.Setenv("PATH", installdir+"/bin:"+installdir+"/sbin:"+path)
	}
	var cmd *exec.Cmd
	env := os.Environ()
	cmd = exec.Command("/bin/bash", "-c", c)
	cmd.Dir = srcdir
	cmd.Env = env
	stdoutStderr, err := cmd.CombinedOutput()
	fmt.Printf("%s\n", stdoutStderr)
	if c != "unbound-anchor" {
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}
	return 0
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
			filename := filepath.Join(path, header.Name)
			info := header.FileInfo()
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
			if !info.ModTime().IsZero() {
				err = os.Chtimes(filename, info.ModTime(), info.ModTime())
				if err != nil {
					return err
				}
			}
		}
	}
	fmt.Println("Done.")
	return nil
}

func main() {
	packages := make(map[Key]string)

	mydir := os.Getenv("PWD")

	// fake root
	installdir := "/Users/jad/setup-getdns-osx-test"
	os.RemoveAll(installdir)

	// build directory - where source code is downloaded to and built
	builddir := "/tmp/"

	// StubbyManager source code
	stubbymgrdir := filepath.Join(builddir, "stubby_manager")
	// Location of files in the final .app bundle
	apppath := filepath.Join(stubbymgrdir, "StubbyManager.app/Contents/MacOS/")

	// source directory for a particular package
	var srcdir string

	path := os.Getenv("PATH")

	// location of qmake - i don't know why i installed it here!
	qmakebin := "/Volumes/JADDevelopment/qt/5.9.1/clang_64/bin/qmake"

	// What to build
	// packagelist := [...]string{"getdns"}
	packagelist := [...]string{"autoconf", "automake", "libtool", "pkgc", "check", "openssl", "libevent", "unbound", "libidn", "getdns"}

	packages[Key{"autoconf", "VERSION"}] = "2.69"
	packages[Key{"autoconf", "DIR"}] = "autoconf-" + packages[Key{"autoconf", "VERSION"}]
	packages[Key{"autoconf", "TAR"}] = packages[Key{"autoconf", "DIR"}] + ".tar.gz"
	packages[Key{"autoconf", "URL"}] = "http://ftpmirror.gnu.org/autoconf/" + packages[Key{"autoconf", "TAR"}]
	packages[Key{"autoconf", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"autoconf", "MAKE_CHECK"}] = ""
	packages[Key{"autoconf", "SIG"}] = ""
	packages[Key{"autoconf", "MAKE"}] = "make -j8"
	packages[Key{"autoconf", "CCACHEOK"}] = "OK"

	packages[Key{"automake", "VERSION"}] = "1.15.1"
	packages[Key{"automake", "DIR"}] = "automake-" + packages[Key{"automake", "VERSION"}]
	packages[Key{"automake", "TAR"}] = packages[Key{"automake", "DIR"}] + ".tar.gz"
	packages[Key{"automake", "URL"}] = "http://ftpmirror.gnu.org/automake/" + packages[Key{"automake", "TAR"}]
	packages[Key{"automake", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"automake", "MAKE_CHECK"}] = ""
	packages[Key{"automake", "SIG"}] = ""
	packages[Key{"automake", "MAKE"}] = "make -j8"
	packages[Key{"automake", "CCACHEOK"}] = "OK"

	packages[Key{"libtool", "VERSION"}] = "2.4.6"
	packages[Key{"libtool", "DIR"}] = "libtool-" + packages[Key{"libtool", "VERSION"}]
	packages[Key{"libtool", "TAR"}] = packages[Key{"libtool", "DIR"}] + ".tar.gz"
	packages[Key{"libtool", "URL"}] = "http://ftpmirror.gnu.org/libtool/" + packages[Key{"libtool", "TAR"}]
	packages[Key{"libtool", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"libtool", "MAKE_CHECK"}] = ""
	packages[Key{"libtool", "SIG"}] = ""
	packages[Key{"libtool", "MAKE"}] = "make -j8"
	packages[Key{"libtool", "CCACHEOK"}] = "OK"

	packages[Key{"pkgc", "VERSION"}] = "0.29.2"
	packages[Key{"pkgc", "DIR"}] = "pkg-config-" + packages[Key{"pkgc", "VERSION"}]
	packages[Key{"pkgc", "TAR"}] = packages[Key{"pkgc", "DIR"}] + ".tar.gz"
	packages[Key{"pkgc", "URL"}] = "https://pkg-config.freedesktop.org/releases/" + packages[Key{"pkgc", "TAR"}]
	packages[Key{"pkgc", "CONFIGURE"}] = "./configure --with-internal-glib --prefix=" + installdir
	packages[Key{"pkgc", "MAKE_CHECK"}] = ""
	packages[Key{"pkgc", "SIG"}] = ""
	packages[Key{"pkgc", "MAKE"}] = "make -j8"
	packages[Key{"pkgc", "CCACHEOK"}] = "OK"

	packages[Key{"check", "VERSION"}] = "0.11.0"
	packages[Key{"check", "DIR"}] = packages[Key{"check", "VERSION"}]
	packages[Key{"check", "TAR"}] = packages[Key{"check", "DIR"}] + ".zip"
	packages[Key{"check", "URL"}] = "https://github.com/libcheck/check/archive/" + packages[Key{"check", "TAR"}]
	packages[Key{"check", "CONFIGURE"}] = "./configure --prefix=" + installdir
	packages[Key{"check", "MAKE_CHECK"}] = ""
	packages[Key{"check", "SIG"}] = ""
	packages[Key{"check", "MAKE"}] = "make -j8"
	packages[Key{"check", "CCACHEOK"}] = "OK"

	packages[Key{"libevent", "VERSION"}] = "2.1.8-stable"
	packages[Key{"libevent", "DIR"}] = "libevent-" + packages[Key{"libevent", "VERSION"}]
	packages[Key{"libevent", "TAR"}] = packages[Key{"libevent", "DIR"}] + ".tar.gz"
	packages[Key{"libevent", "URL_PATH"}] = "release-" + packages[Key{"libevent", "VERSION"}]
	packages[Key{"libevent", "URL"}] = "https://github.com/libevent/libevent/releases/download/" + packages[Key{"libevent", "URL_PATH"}] + "/" + packages[Key{"libevent", "TAR"}]
	packages[Key{"libevent", "CONFIGURE"}] = "./configure --prefix=" + installdir + " CPPFLAGS=-I" + installdir + "/include LDFLAGS=\"-Wl,-headerpad_max_install_names -L" + installdir + "/lib\""
	packages[Key{"libevent", "TEST"}] = "make check"
	packages[Key{"libevent", "SIG"}] = ""
	packages[Key{"libevent", "MAKE"}] = "make -j8"
	packages[Key{"libevent", "CCACHEOK"}] = "OK"

	packages[Key{"openssl", "VERSION"}] = "1.1.0f"
	packages[Key{"openssl", "DIR"}] = "openssl-" + packages[Key{"openssl", "VERSION"}]
	packages[Key{"openssl", "TAR"}] = packages[Key{"openssl", "DIR"}] + ".tar.gz"
	packages[Key{"openssl", "URL"}] = "http://openssl.org/source/" + packages[Key{"openssl", "TAR"}]
	packages[Key{"openssl", "CONFIGURE"}] = "./Configure darwin64-x86_64-cc --shared --openssldir=/Applications/StubbyManager.app/Contents/MacOS/ --prefix=" + installdir + " -Wl,-headerpad_max_install_names"
	packages[Key{"openssl", "MAKE_CHECK"}] = "make test"
	packages[Key{"openssl", "SIG"}] = "9e3e02bc8b4965477a7a1d33be1249299a9deb15"
	packages[Key{"openssl", "MAKE"}] = "make -j8"
	packages[Key{"openssl", "CCACHEOK"}] = "OK"

	packages[Key{"unbound", "VERSION"}] = "1.6.4"
	packages[Key{"unbound", "DIR"}] = "unbound-" + packages[Key{"unbound", "VERSION"}]
	packages[Key{"unbound", "TAR"}] = packages[Key{"unbound", "DIR"}] + ".tar.gz"
	packages[Key{"unbound", "URL"}] = "http://unbound.nlnetlabs.nl/downloads/" + packages[Key{"unbound", "TAR"}]
	packages[Key{"unbound", "CONFIGURE"}] = "patch -lp0 < " + filepath.Join(mydir, "unbound-1.6.3.patch ; ") + "./configure --prefix=" + installdir + " --with-ssl=" + installdir + " --with-conf-file=. LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"unbound", "MAKE_CHECK"}] = "make test"
	packages[Key{"unbound", "SIG"}] = "836ecc48518b9159f600a738c276423ef1f95021"
	packages[Key{"unbound", "MAKE"}] = "make -j8"
	packages[Key{"unbound", "CCACHEOK"}] = "OK"

	packages[Key{"libidn", "VERSION"}] = "1.33"
	packages[Key{"libidn", "DIR"}] = "libidn-" + packages[Key{"libidn", "VERSION"}]
	packages[Key{"libidn", "TAR"}] = packages[Key{"libidn", "DIR"}] + ".tar.gz"
	packages[Key{"libidn", "URL"}] = "http://ftp.gnu.org/gnu/libidn/" + packages[Key{"libidn", "TAR"}]
	packages[Key{"libidn", "CONFIGURE"}] = "./configure --prefix=" + installdir + " LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"libidn", "MAKE_CHECK"}] = "make check"
	packages[Key{"libidn", "SIG"}] = ""
	packages[Key{"libidn", "MAKE"}] = "make -j8"
	packages[Key{"libidn", "CCACHEOK"}] = "OK"

	packages[Key{"getdns", "VERSION"}] = "1.1.2"
	packages[Key{"getdns", "DIR"}] = "getdns-" + packages[Key{"getdns", "VERSION"}]
	packages[Key{"getdns", "URL_PATH"}] = strings.Replace(packages[Key{"getdns", "DIR"}], ".", "-", -1)
	packages[Key{"getdns", "TAR"}] = packages[Key{"getdns", "DIR"}] + ".tar.gz"
	packages[Key{"getdns", "URL"}] = "https://getdnsapi.net/releases/" + packages[Key{"getdns", "URL_PATH"}] + "/" + packages[Key{"getdns", "TAR"}]
	packages[Key{"getdns", "CONFIGURE"}] = "./configure --prefix=" + installdir + " --with-ssl=" + installdir + " --with-libunbound=" + installdir + " --with-libidn=" + installdir + " --with-libevent --enable-debug-daemon LDFLAGS=-Wl,-headerpad_max_install_names"
	packages[Key{"getdns", "MAKE_CHECK"}] = "make check"
	packages[Key{"getdns", "SIG"}] = ""
	packages[Key{"getdns", "MAKE"}] = "make -j8"
	packages[Key{"getdns", "CCACHEOK"}] = "OK"

	// Remove old source code
	for _, v := range packagelist {
		fmt.Println("Cleaning: " + v)
		err := os.RemoveAll(filepath.Join(builddir, packages[Key{v, "DIR"}]))
		if err != nil {
			os.Exit(1)
		}
	}

	// build each package
	for _, v := range packagelist {
		// Is it ok to use ccache?
		ccacheok := true
		if packages[Key{v, "CCACHEOK"}] != "OK" {
			ccacheok = false
		}

		// Are we building check - if so modify the path
		if v == "check" {
			srcdir = filepath.Join(builddir, "check-"+packages[Key{v, "DIR"}])
		} else {
			srcdir = filepath.Join(builddir, packages[Key{v, "DIR"}])
		}
		fmt.Println("Building: " + v)

		// download tarballs or similar
		err := downloadball(builddir, packages[Key{v, "URL"}], packages[Key{v, "TAR"}])
		if err != nil {
			fmt.Println("Download and extract failed " + err.Error())
			os.Exit(1)
		}

		fmt.Println("Configuring: " + v)
		Runcmd(packages[Key{v, "CONFIGURE"}], srcdir, installdir, path, ccacheok)

		fmt.Println("Making: " + v)
		Runcmd(packages[Key{v, "MAKE"}], srcdir, installdir, path, ccacheok)

		fmt.Println("Installing: " + v)
		Runcmd("make install", srcdir, installdir, path, ccacheok)

	}
	err := os.RemoveAll(stubbymgrdir)
	if err != nil {
		os.Exit(1)
	}
	// All packages done - now build StubbyManager.app
	Runcmd("git clone https://jad@portal.sinodun.com/stash/scm/stubui/stubby_manager.git", builddir, builddir, path, true)
	Runcmd("git checkout devel", stubbymgrdir, builddir, path, true)
	Runcmd("patch -p0 < "+filepath.Join(mydir, "/stubbymanager.patch"), stubbymgrdir, builddir, path, true)
	Runcmd(qmakebin+" "+filepath.Join(stubbymgrdir, "/StubbyManager.pro")+" -spec macx-clang CONFIG+=x86_64 && /usr/bin/make qmake_all && make", stubbymgrdir, builddir, path, true)

	fmt.Println("Copying files into .app")
	Runcmd("cp "+installdir+"/bin/stubby "+apppath, srcdir, installdir, path, true)
	Runcmd("cp "+installdir+"/sbin/unbound-anchor "+apppath, srcdir, installdir, path, true)
	dylibs := [...]string{"libgetdns.6.dylib", "libunbound.2.dylib", "libidn.11.dylib", "libssl.1.1.dylib", "libcrypto.1.1.dylib", "libidn.11.dylib"}
	for _, lib := range dylibs {
		Runcmd("cp "+filepath.Join(installdir, "/lib/", lib)+" "+apppath, srcdir, installdir, path, true)
	}
	Runcmd("chmod 755 "+filepath.Join(apppath, "/libssl.1.1.dylib"), srcdir, installdir, path, true)
	Runcmd("chmod 755 "+filepath.Join(apppath, "/libcrypto.1.1.dylib"), srcdir, installdir, path, true)
	for _, lib := range dylibs {
		Runcmd("install_name_tool -id @executable_path/"+lib+" "+lib, apppath, installdir, path, true)
	}
	for _, lib := range dylibs {
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/stubby", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/unbound-anchor", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libgetdns.6.dylib", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libunbound.2.dylib", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libidn.11.dylib", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libssl.1.1.dylib", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libcrypto.1.1.dylib", apppath, installdir, path, true)
		Runcmd("install_name_tool -change "+filepath.Join(installdir, "/lib/", lib)+" @executable_path/"+lib+" "+apppath+"/libidn.11.dylib", apppath, installdir, path, true)
	}
	Runcmd("unbound-anchor", apppath, installdir, "/bin:"+apppath, true)

	fmt.Println("Copying conf files from getdns into the .app directory")
	Runcmd("cp ./getdns-1.1.2/src/tools/stubby.conf "+apppath, builddir, builddir, path, true)
	Runcmd("cp ./getdns-1.1.2/src/tools/stubby.conf "+apppath+"/stubby.conf.default", builddir, builddir, path, true)
	Runcmd("cp ./getdns-1.1.2/src/tools/stubby-setdns-macos.sh "+apppath, builddir, builddir, path, true)
	Writecerts(filepath.Join(apppath, "cert.pem"))
	msg := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>KeepAlive</key>
        <false/>
        <key>Label</key>
        <string>org.getdns.stubby</string>
        <key>LaunchOnlyOnce</key>
        <true/>
        <key>ProgramArguments</key>
        <array>
            <string>/Applications/StubbyManager.app/Contents/MacOS/stubby</string>
            <string>-C</string>
            <string>/Applications/StubbyManager.app/Contents/MacOS/stubby.conf</string>
        </array>
        <key>Sockets</key>
        <dict>
            <key>Listeners</key>
            <dict>
                <key>SockServiceName</key>
                <string>53</string>
                <key>SockType</key>
                <string>dgram</string>
                <key>SockFamily</key>
                <string>IPv4</string>
            </dict>
        </dict>
    </dict>
</plist>`

	f, err := os.Create(filepath.Join(builddir, "org.getdns.stubby.plist"))
	if err != nil {
		fmt.Println("Creating plist file failed: " + err.Error())
		os.Exit(1)
	}
	defer f.Close()
	f.WriteString(string(msg))

	msg = `Now start Packages (http://s.sudre.free.fr/Software/Packages/about.html)
Select Raw package. Then set name and path for package
Add ` + filepath.Join(apppath, "StubbyManager.app") + ` to the payload under /Applications
Add ` + filepath.Join(builddir, "org.getdns.stubby.plist") + ` to the payload under /Library/LaunchDaemons/
`
	fmt.Print(msg)

	os.Exit(0)
}
