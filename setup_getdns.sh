#!/usr/bin/env bash

set -e

PACKAGES=(AUTOCONF AUTOMAKE LIBTOOL PKGC CHECK EXPAT OPENSSL UNBOUND LIBIDN GETDNS)
BUILD_DIR=/tmp
CURL_CMD="/usr/bin/curl -O"
RUN_CHECKS=0
PMAKE=1

usage () {
  echo
  echo "Download and install packages needed to build getdns. Run using sudo if you want to install outside of your home directory"
  echo "Usage: $0 -i <install_directory> options"
  echo "  -c enable make [check|test] when building packages"
  echo "  -d run in debug mode (set -x)"
  echo "  -i <install_directory> Where to install the packages"
  echo "  -j <N> Use parallel make"
  echo "  -h this help!"
  exit 1
}

while getopts ":cdi:j:h" opt; do
    case $opt in
        c  ) RUN_CHECKS=1 ;;
        d  ) set -x ;;
        i  ) INSTALL_DIR=$OPTARG ;;
        j  ) PMAKE=$OPTARG ;;
        h  ) usage ;;
        \? ) usage ;;
    esac
done
[ -z "$INSTALL_DIR" ] && echo "Error: You must specify an install directory." && usage
PATH=$INSTALL_DIR/bin:$INSTALL_DIR/sbin:$PATH

[ $PMAKE -lt 1 ] && echo "Error: Parallel make must be > 0." && usage

AUTOCONF_VERSION=2.69
AUTOCONF_DIR=autoconf-$AUTOCONF_VERSION
AUTOCONF_TAR=$AUTOCONF_DIR.tar.gz
AUTOCONF_URL=" -L http://ftpmirror.gnu.org/autoconf/autoconf-${AUTOCONF_VERSION}.tar.gz"
AUTOCONF_CONFIGURE="./configure --prefix=$INSTALL_DIR"
AUTOCONF_MAKE_CHECK=""
AUTOCONF_SIG=""

AUTOMAKE_VERSION=1.15
AUTOMAKE_DIR=automake-$AUTOMAKE_VERSION
AUTOMAKE_TAR=$AUTOMAKE_DIR.tar.gz
AUTOMAKE_URL=" -L http://ftpmirror.gnu.org/automake/automake-${AUTOMAKE_VERSION}.tar.gz"
AUTOMAKE_CONFIGURE="./configure --prefix=$INSTALL_DIR"
AUTOMAKE_MAKE_CHECK=""
AUTOMAKE_SIG=""

LIBTOOL_VERSION=2.4.6
LIBTOOL_DIR=libtool-$LIBTOOL_VERSION
LIBTOOL_TAR=$LIBTOOL_DIR.tar.gz
LIBTOOL_URL=" -L http://ftpmirror.gnu.org/libtool/libtool-${LIBTOOL_VERSION}.tar.gz"
LIBTOOL_CONFIGURE="./configure --prefix=$INSTALL_DIR"
LIBTOOL_MAKE_CHECK=""
LIBTOOL_SIG=""

PKGC_VERSION=0.29.2
PKGC_DIR=pkg-config-$PKGC_VERSION
PKGC_TAR=$PKGC_DIR.tar.gz
PKGC_URL=" -L https://pkg-config.freedesktop.org/releases/pkg-config-${PKGC_VERSION}.tar.gz"
PKGC_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-internal-glib"
PKGC_MAKE_CHECK=""
PKGC_SIG=""

CHECK_VERSION=0.11.0
CHECK_DIR=$CHECK_VERSION
CHECK_TAR=${CHECK_DIR}.zip
CHECK_URL=" -L https://github.com/libcheck/check/archive/${CHECK_VERSION}.zip"
CHECK_CONFIGURE="./configure --prefix=$INSTALL_DIR"
CHECK_MAKE_CHECK="" # skip as very slow
CHECK_SIG="" # I am not aware of a sig for this tarball

EXPAT_VERSION=2.1.0
EXPAT_DIR=expat-$EXPAT_VERSION
EXPAT_TAR=$EXPAT_DIR.tar.gz
EXPAT_URL=" -L http://downloads.sourceforge.net/project/expat/expat/${EXPAT_VERSION}/expat-${EXPAT_VERSION}.tar.gz"
EXPAT_CONFIGURE="./configure --prefix=$INSTALL_DIR"
EXPAT_MAKE_CHECK=""
EXPAT_SIG="" # I am not aware of a sig for this tarball

OPENSSL_VERSION=1.1.0f
OPENSSL_DIR=openssl-${OPENSSL_VERSION}
OPENSSL_TAR=${OPENSSL_DIR}.tar.gz
OPENSSL_URL=" -L http://openssl.org/source/${OPENSSL_TAR}"
OPENSSL_CONFIGURE="./Configure darwin64-x86_64-cc --shared --prefix=$INSTALL_DIR"
OPENSSL_MAKE_CHECK="make test"
OPENSSL_SIG="9e3e02bc8b4965477a7a1d33be1249299a9deb15"

UNBOUND_VERSION=1.6.3
UNBOUND_DIR=unbound-${UNBOUND_VERSION}
UNBOUND_TAR=${UNBOUND_DIR}.tar.gz
UNBOUND_URL=http://unbound.nlnetlabs.nl/downloads/${UNBOUND_TAR}
UNBOUND_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-libexpat=$INSTALL_DIR"
UNBOUND_MAKE_CHECK="make test"
UNBOUND_SIG="4477627c31e8728058565f3bae3a12a1544d8a9c"

LIBIDN_VERSION=1.33
LIBIDN_DIR=libidn-${LIBIDN_VERSION}
LIBIDN_TAR=${LIBIDN_DIR}.tar.gz
LIBIDN_URL=http://ftp.gnu.org/gnu/libidn/${LIBIDN_TAR}
LIBIDN_CONFIGURE="./configure --prefix=$INSTALL_DIR"
LIBIDN_MAKE_CHECK="make check"
LIBIDN_SIG="" # Uses GPG TODO...

GETDNS_VERSION=1.1.1
GETDNS_DIR=getdns-${GETDNS_VERSION}
GETDNS_URL_PATH=${GETDNS_DIR//./-}
GETDNS_TAR=${GETDNS_DIR}.tar.gz
GETDNS_URL=" -L https://getdnsapi.net/releases/${GETDNS_URL_PATH}/${GETDNS_TAR}"
GETDNS_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-libunbound=$INSTALL_DIR --with-libidn=$INSTALL_DIR --enable-debug-daemon"
GETDNS_MAKE_CHECK="make check"
GETDNS_SIG=""

for PACKAGE in ${PACKAGES[*]} ; do
  cd $BUILD_DIR
  if eval [ ! -n \"\$${PACKAGE}_VERSION\" ] || \
     eval [ ! -n \"\$${PACKAGE}_DIR\" ] || \
     eval [ ! -n \"\$${PACKAGE}_TAR\" ] || \
     eval [ ! -n \"\$${PACKAGE}_URL\" ] || \
     eval [ ! -n \"\$${PACKAGE}_CONFIGURE\" ] ; then 
    echo "Package ${PACKAGE} spec not found"
    exit 1
  fi
  eval [ -d \$${PACKAGE}_DIR ] && eval rm -rf \$${PACKAGE}_DIR
  eval [ ! -e \$${PACKAGE}_TAR ] && eval ${CURL_CMD} \$${PACKAGE}_URL
  if eval [ -n \"\$${PACKAGE}_SIG\" ] ; then
    eval SIG=\$\(shasum \$${PACKAGE}_TAR \| awk \' { print \$1 } \' \)
    if eval [ x_$SIG != x_\$${PACKAGE}_SIG ] ; then
      echo "Package ${PACKAGE} has a bad signature" 
      exit 1
    fi
  fi
  if eval [ -e \$${PACKAGE}_TAR ] ; then
          if eval [[ "\$${PACKAGE}_TAR" =~ .\*.tar.\* ]] ; then
                  eval tar -xf \$${PACKAGE}_TAR
          elif eval [[ \$${PACKAGE}_TAR =~ .\*.zip ]] ; then
                  eval unzip \$${PACKAGE}_TAR && mv $BUILD_DIR/check-${CHECK_DIR} $BUILD_DIR/${CHECK_DIR}
          fi
  fi
  eval cd \$${PACKAGE}_DIR
  if [ $PACKAGE == "CHECK" ] && [ ! -x configure ] ; then
          autoreconf -i
  fi
  eval \$${PACKAGE}_CONFIGURE
  PMAKE2=$PMAKE
  [ $PACKAGE == "OPENSSL" ] && PMAKE2=1
  make -j $PMAKE2
  [ $RUN_CHECKS -eq 1 ] && eval "\$${PACKAGE}_MAKE_CHECK"
  make install -j $PMAKE2
  
done
