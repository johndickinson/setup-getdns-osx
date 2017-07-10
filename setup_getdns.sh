#!/usr/bin/env bash

set -e

# PACKAGES=(AUTOCONF AUTOMAKE LIBTOOL PKGC CHECK OPENSSL LIBEVENT UNBOUND LIBIDN GETDNS)
PACKAGES=(LIBEVENT)

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
rm -rf ${INSTALL_DIR}
APP_PATH=${INSTALL_DIR}/Stubby.app/Contents/MacOS/
mkdir -p ${APP_PATH}
LOG_PATH=${INSTALL_DIR}/logs
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

OPENSSL_VERSION=1.1.0f
OPENSSL_DIR=openssl-${OPENSSL_VERSION}
OPENSSL_TAR=${OPENSSL_DIR}.tar.gz
OPENSSL_URL=" -L http://openssl.org/source/${OPENSSL_TAR}"
OPENSSL_CONFIGURE="./Configure darwin64-x86_64-cc --shared --prefix=$INSTALL_DIR -Wl,-headerpad_max_install_names"
OPENSSL_MAKE_CHECK="make test"
OPENSSL_SIG="9e3e02bc8b4965477a7a1d33be1249299a9deb15"

OPENSSL102_VERSION=1.0.2l
OPENSSL102_DIR=openssl-${OPENSSL102_VERSION}
OPENSSL102_TAR=${OPENSSL102_DIR}.tar.gz
OPENSSL102_URL=" -L http://openssl.org/source/${OPENSSL102_TAR}"
OPENSSL102_CONFIGURE="./Configure darwin64-x86_64-cc --shared --prefix=$INSTALL_DIR -Wl,-headerpad_max_install_names"
OPENSSL102_MAKE_CHECK="make test"
OPENSSL102_SIG="b58d5d0e9cea20e571d903aafa853e2ccd914138"

LIBEVENT_VERSION=2.1.8-stable
LIBEVENT_DIR=libevent-${LIBEVENT_VERSION}
LIBEVENT_TAR=${LIBEVENT_DIR}.tar.gz
LIBEVENT_URL_PATH="release-${LIBEVENT_VERSION}-stable"
LIBEVENT_URL=" -L https://github.com/libevent/libevent/releases/download/${LIBEVENT_URL_PATH}/${LIBEVENT_TAR}"
LIBEVENT_CONFIGURE="./configure --prefix=$INSTALL_DIR CPPFLAGS=-I$INSTALL_DIR/include LDFLAGS=\\\"-Wl,-headerpad_max_install_names -L$INSTALL_DIR/lib\\\""
LIBEVENT_MAKE_CHECK="make check"
LIBEVENT_SIG=""

UNBOUND_VERSION=1.6.3
UNBOUND_DIR=unbound-${UNBOUND_VERSION}
UNBOUND_TAR=${UNBOUND_DIR}.tar.gz
UNBOUND_URL=http://unbound.nlnetlabs.nl/downloads/${UNBOUND_TAR}
UNBOUND_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-conf-file=. LDFLAGS=-Wl,-headerpad_max_install_names"
UNBOUND_MAKE_CHECK="make test"
UNBOUND_SIG="4477627c31e8728058565f3bae3a12a1544d8a9c"

LIBIDN_VERSION=1.33
LIBIDN_DIR=libidn-${LIBIDN_VERSION}
LIBIDN_TAR=${LIBIDN_DIR}.tar.gz
LIBIDN_URL=http://ftp.gnu.org/gnu/libidn/${LIBIDN_TAR}
LIBIDN_CONFIGURE="./configure --prefix=$INSTALL_DIR LDFLAGS=-Wl,-headerpad_max_install_names"
LIBIDN_MAKE_CHECK="make check"
LIBIDN_SIG="" # Uses GPG TODO...

GETDNS_VERSION=1.1.1
GETDNS_DIR=getdns-${GETDNS_VERSION}
GETDNS_URL_PATH=${GETDNS_DIR//./-}
GETDNS_TAR=${GETDNS_DIR}.tar.gz
GETDNS_URL=" -L https://getdnsapi.net/releases/${GETDNS_URL_PATH}/${GETDNS_TAR}"
GETDNS_CONFIGURE="./configure --prefix=$INSTALL_DIR --with-ssl=$INSTALL_DIR --with-libunbound=$INSTALL_DIR --with-libidn=$INSTALL_DIR --with-libevent --enable-debug-daemon LDFLAGS=-Wl,-headerpad_max_install_names"
GETDNS_MAKE_CHECK="make check"
GETDNS_SIG=""

export LDFLAGS=-Wl,-headerpad_max_install_names

for PACKAGE in ${PACKAGES[*]} ; do
  mkdir -p ${LOG_PATH}/${PACKAGE}
  echo "$(date) Building: ${PACKAGE}"
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
  eval [ ! -e \$${PACKAGE}_TAR ] && eval ${CURL_CMD} \$${PACKAGE}_URL > ${LOG_PATH}/${PACKAGE}/download.log 2>&1
  if eval [ -n \"\$${PACKAGE}_SIG\" ] ; then
    eval SIG=\$\(shasum \$${PACKAGE}_TAR \| awk \' { print \$1 } \' \)
    if eval [ x_$SIG != x_\$${PACKAGE}_SIG ] ; then
      echo "Package ${PACKAGE} has a bad signature"
      exit 1
    fi
  fi
  eval rm -rf \$${PACKAGE}_DIR
  if eval [ -e \$${PACKAGE}_TAR ] ; then
    if eval [[ "\$${PACKAGE}_TAR" =~ .\*.tar.\* ]] ; then
      eval tar -xf \$${PACKAGE}_TAR
    elif eval [[ \$${PACKAGE}_TAR =~ .\*.zip ]] ; then
      eval unzip -qq \$${PACKAGE}_TAR && mv $BUILD_DIR/check-${CHECK_DIR} $BUILD_DIR/${CHECK_DIR} 
    fi
  fi
  eval cd \$${PACKAGE}_DIR
  if [ ${PACKAGE} == "UNBOUND" ] ; then
    patch -lp0 < ../unbound-1.6.3.patch
  fi
  if [ $PACKAGE == "CHECK" ] && [ ! -x configure ] ; then
    autoreconf -i > ${LOG_PATH}/${PACKAGE}/autoreconf.log 2>&1
  fi
  eval \$${PACKAGE}_CONFIGURE > ${LOG_PATH}/${PACKAGE}/configure.log 2>&1
  make -j $PMAKE > ${LOG_PATH}/${PACKAGE}/make.log 2>&1
  [ $RUN_CHECKS -eq 1 ] && eval "\$${PACKAGE}_MAKE_CHECK" > ${LOG_PATH}/${PACKAGE}/make-check.log 2>&1
  make install -j $PMAKE > ${LOG_PATH}/${PACKAGE}/make-install.log 2>&1
  
done

echo "$(date) Copying files into .app"
cp ${INSTALL_DIR}/bin/stubby ${APP_PATH}/stubby
cp ${INSTALL_DIR}/sbin/unbound-anchor ${APP_PATH}/unbound-anchor
echo "$(date) Running install_name_tool"
dylibs="libgetdns.6.dylib libunbound.2.dylib libidn.11.dylib libssl.1.0.0.dylib libcrypto.1.0.0.dylib libidn.11.dylib"
for dylib in ${dylibs} ; do
  cp ${INSTALL_DIR}/lib/${dylib} ${APP_PATH}/${dylib}
done
chmod 755 ${APP_PATH}/libssl.1.0.0.dylib
chmod 755 ${APP_PATH}/libcrypto.1.0.0.dylib
for dylib in ${dylibs} ; do
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/stubby
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/unbound-anchor
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libgetdns.6.dylib
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libunbound.2.dylib
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libidn.11.dylib
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libssl.1.0.0.dylib
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libcrypto.1.0.0.dylib
  install_name_tool -change ${INSTALL_DIR}/lib/${dylib} @executable_path/${dylib} ${APP_PATH}/libidn.11.dylib
done
cd ${APP_PATH}
./unbound-anchor