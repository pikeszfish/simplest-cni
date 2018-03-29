#!/usr/bin/env bash

# Mount Usages(required):
# /etc/cni/net.d:/etc/cni/net.d
# /opt/cni/bin:/host/opt/cni/bin

# - Expects the host CNI binary path to be mounted at /host/opt/cni/bin.
# - Expects the host CNI network config path to be mounted at /etc/cni/net.d.

set -ex

HOST_CNI_NET_DIR=${CNI_NET_DIR:-/etc/cni/net.d}

# /simplest-cni-config for ConfigMap
CONFIG_MOUNT_DIR=${CONFIG_MOUNT_DIR:-/simplest-cni-config}

# Delete all old host path bin
rm -f /host/opt/cni/bin/bridge
rm -f /host/opt/cni/bin/loopback
rm -f /host/opt/cni/bin/host-local
rm -f /host/opt/cni/bin/portmap

HOST_BIN_PATH="/host/opt/cni/bin"

if [ ! -w ${HOST_BIN_PATH} ];
then
	echo "${HOST_BIN_PATH} is non-writeable"
	exit 1
fi

# cp all the bin needed
for path in /opt/cni/bin/*;
do
	filename="$(basename $path)"

	cp $path ${HOST_BIN_PATH}
	if [ "$?" != "0" ];
	then
		echo "Failed to copy $path to ${HOST_BIN_PATH}. This may be caused by selinux configuration on the host, or something else."
		exit 1
	fi
done


# This funtion was used for reload host /etc/cni/net.d/10-bridge.conflist
# cni config template from ConfigMap /hly-config/cni_network_config
# config from ConfigMap /hly-config/xxx
# cert/key/ca are just paths not whole file
function reload_cni_config_file() {
    # Copy form configMap
    TMP_CONF="/10-bridge.conflist.tmp"
    cp ${CONFIG_MOUNT_DIR}/cni_network_config ${TMP_CONF}

    # cat config from configMap
    MTU=`cat ${CONFIG_MOUNT_DIR}/mtu`
    BRIDGE=`cat ${CONFIG_MOUNT_DIR}/bridge`
    IS_GATEWAY=`cat ${CONFIG_MOUNT_DIR}/is_gateway`
    IS_DEFAULT_GATEWAY=`cat ${CONFIG_MOUNT_DIR}/is_default_gateway`
    FORCE_ADDRESS=`cat ${CONFIG_MOUNT_DIR}/force_address`
    IP_MASQ=`cat ${CONFIG_MOUNT_DIR}/ip_masq`
    HAIRPIN_MODE=`cat ${CONFIG_MOUNT_DIR}/hairpin_mode`
    PROMISC_MODE=`cat ${CONFIG_MOUNT_DIR}/promisc_mode`
    DATA_DIR=`cat ${CONFIG_MOUNT_DIR}/data_dir`
    
    mkdir -p ${DATA_DIR}

    sed -i s~__MTU__~${MTU:-false}~g $TMP_CONF
    sed -i s~__BRIDGE__~${BRIDGE:-br0}~g $TMP_CONF
    sed -i s~__IS_GATEWAY__~${IS_GATEWAY:-false}~g $TMP_CONF
    sed -i s~__IS_DEFAULT_GATEWAY__~${IS_DEFAULT_GATEWAY:-false}~g $TMP_CONF
    sed -i s~__FORCE_ADDRESS__~${FORCE_ADDRESS:-false}~g $TMP_CONF
    sed -i s~__IP_MASQ__~${IP_MASQ:-false}~g $TMP_CONF
    sed -i s~__HAIRPIN_MODE__~${HAIRPIN_MODE:-false}~g $TMP_CONF
    sed -i s~__PROMISC_MODE__~${PROMISC_MODE:-false}~g $TMP_CONF
    sed -i s~__DATA_DIR__~${DATA_DIR:-/tmp/k8s-cni-local-host}~g $TMP_CONF

    CNI_CONF_NAME=${CNI_CONF_NAME:-10-bridge.conflist}
    CNI_OLD_CONF_NAME=${CNI_OLD_CONF_NAME:-10-bridge.conflist}

    echo "CNI config: $(cat ${TMP_CONF})"

    for old_conf_file in `echo ${CNI_OLD_CONF_NAME} | sed "s/,/ /g"`;
    do
        rm -f "/etc/cni/net.d/${old_conf_file}"
    done

    # install CNI config to host path
    mv $TMP_CONF /etc/cni/net.d/${CNI_CONF_NAME}
    if [ "$?" != "0" ];
    then
        echo "Failed to mv files. This may be caused by selinux configuration on the host, or something else."
        exit 1
    fi

    echo "Created CNI config ${CNI_CONF_NAME}"
}

reload_cni_config_file

should_sleep=${SLEEP:-"true"}

old_stat_config=`stat -c%y ${CONFIG_MOUNT_DIR} 2>/dev/null`

while [ "$should_sleep" == "true"  ]; do
    current_stat_config=`stat -c%y ${CONFIG_MOUNT_DIR} 2>/dev/null`

    if [ "${current_stat_config}" != "${old_stat_config}" ];
    then
        echo "Updating installed config at: $(date)"
        old_stat_config=${current_stat_config}
        reload_cni_config_file
    fi
    sleep 30
done
