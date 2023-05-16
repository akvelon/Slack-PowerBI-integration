#! /usr/bin/env bash

set -o errexit -o noclobber -o nounset -o pipefail

function __get_deployment_status() {
  local deployment_id="${1}"
  local deployment_status
  deployment_status="$(aws deploy get-deployment --deployment-id "${deployment_id}" | jq --raw-output '.deploymentInfo.status')"
  echo -n "${deployment_status}"
}

function get_deployment_status() {
  [[ "${#}" -ne 1 ]] && { echo "${FUNCNAME[0]}: missing arguments: <DEPLOYMENT_ID>" 1>&2; return 1; }

  local deployment_id="${1}"
  __get_deployment_status "${deployment_id}"
}

function __wait_deployment_completion() {
  local deployment_id="${1}"
  sleep 20s
  local deployment_status
  deployment_status="$(get_deployment_status "${deployment_id}")"
  echo "info: deployment_status=${deployment_status}"
  if [[ ! "${deployment_status}" == 'Failed' ]] && [[ ! "${deployment_status}" == 'Succeeded' ]]; then
    __wait_deployment_completion "${deployment_id}"
  fi

  if [[ "${deployment_status}" == 'Failed' ]]; then
    echo "error: deployment_status=${deployment_status} deployment_id=${deployment_id}"

    return 2
  fi
}

function wait_deployment_completion() {
  [[ "${#}" -ne 1 ]] && { echo "${FUNCNAME[0]}: missing arguments: <DEPLOYMENT_ID>" 1>&2; return 1; }

  local deployment_id="${1}"
  __wait_deployment_completion "${deployment_id}"
}

function __deploy_to_ec2() {
  local environment="${1}"
  local revision_bucket="${2}"
  local application_name="${3}"
  local revision_description="${4}"
  local deployment_group_name="${application_name}-${environment}"
  local s3_location="bucket=${revision_bucket},bundleType=tgz,key=${application_name}.tgz"
  local deployment_id
  deployment_id="$(aws deploy create-deployment --application-name "${application_name}" --deployment-group-name "${deployment_group_name}" --description "${revision_description}" --s3-location "${s3_location}" | jq --raw-output '.deploymentId')"
  echo "info: deployment_id=${deployment_id}"
  wait_deployment_completion "${deployment_id}"
}

function deploy_to_ec2() {
  [[ "${#}" -ne 4 ]] && { echo "${FUNCNAME[0]}: missing arguments: <ENVIRONMENT> <REVISION_BUCKET> <APPLICATION_NAME> <REVISION_DESCRIPTION>" 1>&2; return 1; }

  local environment="${1}"
  local revision_bucket="${2}"
  local application_name="${3}"
  local revision_description="${4}"
  __deploy_to_ec2 "${environment}" "${revision_bucket}" "${application_name}" "${revision_description}"
}
