#!/bin/bash
set -euo pipefail

mkdir -p "$(dirname "$SLSA_OUTPUTS_ARTIFACTS_FILE")"

subjects=""
for binary in "$BUILD_DIR/avalanchego" "$BUILD_DIR/plugins/evm"; do
  if [ -f "$binary" ]; then
    hash=$(sha256sum "$binary" | awk '{print $1}')
    subject_name=$(basename "$binary")
    
    [ -n "$subjects" ] && subjects+=","
    
    printf -v subject '{"name": "%s", "digest": {"sha256": "%s"}}' "$subject_name" "$hash"
    subjects+="$subject"
  fi
done

cat <<EOF > "$SLSA_OUTPUTS_ARTIFACTS_FILE"
{
  "version": 1,
  "attestations": [{
    "name": "$BINARY_NAME-binaries",
    "subjects": [${subjects}]
  }]
}
EOF

cat "$SLSA_OUTPUTS_ARTIFACTS_FILE"