#!/usr/bin/env bash

set -e

PACKAGES=(AUTOCONF AUTOMAKE LIBTOOL GDB CHECK EXPAT OPENSSL UNBOUND LIBIDN LDNS)
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

AUTOMAKE_VERSION=1.14
AUTOMAKE_DIR=automake-$AUTOMAKE_VERSION
AUTOMAKE_TAR=$AUTOMAKE_DIR.tar.gz
AUTOMAKE_URL=" -L http://ftpmirror.gnu.org/automake/automake-${AUTOMAKE_VERSION}.tar.gz"
AUTOMAKE_CONFIGURE="./configure --prefix=$INSTALL_DIR"
AUTOMAKE_MAKE_CHECK=""
AUTOMAKE_SIG=""

LIBTOOL_VERSION=2.4.2
LIBTOOL_DIR=libtool-$LIBTOOL_VERSION
LIBTOOL_TAR=$LIBTOOL_DIR.tar.gz
LIBTOOL_URL=" -L http://ftpmirror.gnu.org/libtool/libtool-${LIBTOOL_VERSION}.tar.gz"
LIBTOOL_CONFIGURE="./configure --prefix=$INSTALL_DIR"
LIBTOOL_MAKE_CHECK=""
LIBTOOL_SIG=""

GDB_VERSION=7.9
GDB_DIR=gdb-$GDB_VERSION
GDB_TAR=$GDB_DIR.tar.gz
GDB_URL=" -L http://ftp.gnu.org/gnu/gdb/$GDB_TAR"
GDB_CONFIGURE="./configure --prefix=$INSTALL_DIR"
GDB_MAKE_CHECK=""
GDB_SIG=""

CHECK_VERSION=0.9.14
CHECK_DIR=check-$CHECK_VERSION
CHECK_TAR=${CHECK_DIR}.tar.gz
CHECK_URL=" -L http://downloads.sourceforge.net/project/check/check/${CHECK_VERSION}/check-${CHECK_VERSION}.tar.gz"
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

OPENSSL_VERSION=1.0.2d
OPENSSL_DIR=openssl-${OPENSSL_VERSION}
OPENSSL_TAR=${OPENSSL_DIR}.tar.gz
OPENSSL_URL=http://openssl.org/source/${OPENSSL_TAR}
OPENSSL_CONFIGURE="./Configure darwin64-x86_64-cc --shared --prefix=$INSTALL_DIR"
OPENSSL_MAKE_CHECK="make test"
OPENSSL_SIG="d01d17b44663e8ffa6a33a5a30053779d9593c3d"

UNBOUND_VERSION=1.5.4
UNBOUND_DIR=unbound-${UNBOUND_VERSION}
UNBOUND_TAR=${UNBOUND_DIR}.tar.gz
UNBOUND_URL=http://unbound.nlnetlabs.nl/downloads/${UNBOUND_TAR}
UNBOUND_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-libexpat=$INSTALL_DIR"
UNBOUND_MAKE_CHECK="make test"
UNBOUND_SIG="ce0abc1563baa776a0f2c21516ffc13e6bff7d0f"

LIBIDN_VERSION=1.30
LIBIDN_DIR=libidn-${LIBIDN_VERSION}
LIBIDN_TAR=${LIBIDN_DIR}.tar.gz
LIBIDN_URL=http://ftp.gnu.org/gnu/libidn/${LIBIDN_TAR}
LIBIDN_CONFIGURE="./configure --prefix=$INSTALL_DIR"
LIBIDN_MAKE_CHECK="make check"
LIBIDN_SIG="" # Uses GPG TODO...

LDNS_VERSION=1.6.17
LDNS_DIR=ldns-${LDNS_VERSION}
LDNS_TAR=${LDNS_DIR}.tar.gz
LDNS_URL=http://nlnetlabs.nl/downloads/ldns/${LDNS_TAR}
LDNS_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-drill"
LDNS_MAKE_CHECK="" # No tests
LDNS_SIG="4218897b3c002aadfc7280b3f40cda829e05c9a4"

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
  eval [ ! -e \$${PACKAGE}_TAR ] && eval ${CURL_CMD} -O \$${PACKAGE}_URL
  if eval [ -n \"\$${PACKAGE}_SIG\" ] ; then
    eval SIG=\$\(shasum \$${PACKAGE}_TAR \| awk \' { print \$1 } \' \)
    if eval [ x_$SIG != x_\$${PACKAGE}_SIG ] ; then
      echo "Package ${PACKAGE} has a bad signature" 
      exit 1
    fi
  fi
  eval tar -xf \$${PACKAGE}_TAR
  eval cd \$${PACKAGE}_DIR
  eval \$${PACKAGE}_CONFIGURE
  PMAKE2=$PMAKE
  [ $PACKAGE == "OPENSSL" ] && PMAKE2=1
  make -j $PMAKE2
  [ $RUN_CHECKS -eq 1 ] && eval "\$${PACKAGE}_MAKE_CHECK"
  make install -j $PMAKE2
  
done
echo
echo "Now clone your GetDNS repo and build it like this:"
echo "cd getdns"
echo "export PATH=$INSTALL_DIR:\$PATH"
echo "autoreconf --install"
echo "./configure --prefix=$INSTALL_DIR \\"
echo "            --with-ssl=$INSTALL_DIR \\"
echo "            --with-libunbound=$INSTALL_DIR \\"
echo "            --with-libidn=$INSTALL_DIR \\"
echo "            --with-libldns=$INSTALL_DIR"
echo "make"
echo "make install"