#!/bin/bash
set -eu

export PATH=/usr/bin:/bin:/usr/sbin:/sbin

script_name="$(basename ${BASH_SOURCE[0]})"

# supported tools
telemetry_tools=(
	authenticator
	clientds
	example
	generator
	help
)

usage()
{
	local exit_status=${1:-0}

	cat - << _EOF_
Usage:
	${script_name} <tool> <args>...
Where:
	<tool>
		is one of: ${telemetry_tools[*]}
	<args>...
		are the arguments to pass to that tool
Examples:
	${script_name} example
	${script_name} generator --debug
Notes:
  * If no <tool> is specified then this help message is shown.
  * Any certs that are found in the directory specified by the
    TELEMETRY_CERTS_DIR env var will be added to the system
    certs store.
  * An appropriate config file, specified by the TELEMETRY_CONFIG
    env var, will be generated using the TELEMETRY_* env vars,
    and automatically specified via a --config <config> option to
    relevant tools.
  * Appropriate options and arguments will be added to the provided
    <args> depending on the tool the has been specified. For example
    a --telemetry <telemetry_type> option and a <telemetry_json>
    argument will be added to the generator tool's command line,
    using the values specified by the associated TELEMETRY_TYPE and
    TELEMETRY_JSON env vars.
_EOF_
	exit ${exit_status}
}

# this entrypoint script must run as root
if [[ "$(id -u)" != "0" ]]; then
	echo "Error: entrypoint script must run as root"
	usage 1
fi

# at least one argument, specifying the tool to run, must be provided
if (( $# < 1 )); then
	echo "Error: first argument must specify the tool to run"
	usage 1
fi

# save the selected tool name
tool="${1}"

# remaining arguments will be passed to the tool
shift

# verify that the select tool is valid
case "${tool}" in
	(help)
		usage 0
		;;
	(authenticator|clientds|generator)
		cmd="/usr/bin/telemetry-${tool}"
		;;
	(example)
		cmd="/app/${tool}"
		;;
	(*)
		echo "Error: invalid tool '${tool}'"
		usage 1
		;;
esac


# config environment vars
user="${TELEMETRY_USER:-susetelm}"
group="${TELEMETRY_GROUP:-susetelm}"
config="${TELEMETRY_CONFIG:-/etc/susetelemetry/telemetry.yaml}"
certs_dir="${TELEMETRY_CERTS_DIR:-${HOME}/certs}"
base_url="${TELEMETRY_BASE_URL:-http://localhost:9999/telemetry}"
enabled="${TELEMETRY_ENABLED:-enabled}"
client_id="${TELEMETRY_CLIENT_ID:-}"
customer_id="${TELEMETRY_CUSTOMER_ID:-}"
ds_driver="${TELEMETRY_DATASTORE_DRIVER:-sqlite3}"
ds_params="${TELEMETRY_DATASTORE_PARAMS:-${HOME}/data/telemetry.db}"
log_level="${TELEMETRY_LOG_LEVEL:-info}"

# argument env vars
telemetry_type="${TELEMETRY_TYPE:-TEST-TELEMETRY-SERVICE}"
telemetry_json="${TELEMETRY_JSON:-/app/data/blob.json}"

# derived vars
case "${ds_params}" in
(*:memory:*)
	ds_dir=""
	;;
(*/*)
	ds_dir="${ds_params%/*}"
	;;
(*)
	ds_dir="."
	;;
esac

# verify that the datastore directory exists if needed
if [[ -n "${ds_dir}" ]] && [[ ! -d "${ds_dir}" ]]; then
    echo "Error: '${ds_dir}' directory not found"
    usage 1
fi

# ensure the generator command exists
if [[ ! -x "${cmd}" ]]; then
	echo "Error: '${cmd}' command not found"
	usage 1
fi

# if a config file doesn't already exist, generate a telemetry
# config based upon env settings
if [[ ! -e "${config}" ]]; then
	config_dir="$(dirname "${config}")"
	if [[ ! -d "${config_dir}" ]]; then
		echo "Creating config directory '${config_dir}'"
		mkdir -m 755 -p ${config_dir}
		chown ${user}:${group} ${config_dir}
	fi
	echo "Generating config '${config}'"
	cat - > "${config}" << _EOF_
telemetry_base_url: ${base_url}
enabled: ${enabled}
client_id: ${client_id}
customer_id: ${customer_id}
tags: []
datastores:
  driver: ${ds_driver}
  params: "${ds_params}"
class_options:
  opt_out: true
  opt_in: false
  allow: []
  deny: []
logging:
  level: ${log_level}
  location: stderr
  style: text
_EOF_
	chmod 644 "${config}"
	chown ${user}:${group} "${config}"
fi

# install additional certs if provided
if [[ -d "${certs_dir}" ]]; then
	certs=( $(ls -1 ${certs_dir}) )
	if (( ${#certs[@]} )); then
		cp ${certs_dir}/* /etc/pki/trust/anchors
		update-ca-certificates
	fi
fi

# construct the command line to be executed
cmd_args=(
	"${cmd}"
	"--config"
	"${config}"
)

# add appropriate args for the specified tool
case "${tool}" in
	(generator)
		cmd_args+=(
			"--telemetry"
			"${telemetry_type}"
			"${@}"
			"${telemetry_json}"
		)
		;;
	(*)
		cmd_args+=( "${@}" )
		;;
esac

# exec specified command as specified user
echo "Running as ${user}: ${cmd_args[*]}"
exec runuser -u "${user}" -- "${cmd_args[@]}"
