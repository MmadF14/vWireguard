#!/usr/bin/env bash
set -e

DIR=$(dirname "$0")

# install node modules
YARN=yarn
[ -x /usr/bin/lsb_release ] && [ -n "`lsb_release -i | grep Debian`" ] && YARN=yarnpkg
$YARN install

# Create necessary directories
mkdir -p "${DIR}/assets/dist/js" "${DIR}/assets/dist/css"

# Copy admin-lte dist
cp -r "${DIR}/node_modules/admin-lte/dist/js/adminlte.min.js" "${DIR}/assets/dist/js/adminlte.min.js"
cp -r "${DIR}/node_modules/admin-lte/dist/css/adminlte.min.css" "${DIR}/assets/dist/css/adminlte.min.css"

# Copy custom files
cp -r "${DIR}/static/dist/css/custom.css" "${DIR}/assets/dist/css/custom.css"
cp -r "${DIR}/static/dist/js/custom.js" "${DIR}/assets/dist/js/custom.js"

# Copy helper js
cp -r "${DIR}/custom" "${DIR}/assets"

# Copy plugins
mkdir -p "${DIR}/assets/plugins"
cp -r "${DIR}/node_modules/admin-lte/plugins/jquery" \
  "${DIR}/node_modules/admin-lte/plugins/fontawesome-free" \
  "${DIR}/node_modules/admin-lte/plugins/bootstrap" \
  "${DIR}/node_modules/admin-lte/plugins/icheck-bootstrap" \
  "${DIR}/node_modules/admin-lte/plugins/toastr" \
  "${DIR}/node_modules/admin-lte/plugins/jquery-validation" \
  "${DIR}/node_modules/admin-lte/plugins/select2" \
  "${DIR}/node_modules/jquery-tags-input" \
  "${DIR}/assets/plugins/"

# Create static directory structure
mkdir -p "${DIR}/static/dist/js" "${DIR}/static/dist/css"

# Copy assets to static directory
cp -r "${DIR}/assets/dist/js/"* "${DIR}/static/dist/js/"
cp -r "${DIR}/assets/dist/css/"* "${DIR}/static/dist/css/"
cp -r "${DIR}/assets/plugins" "${DIR}/static/"
