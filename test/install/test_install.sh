#!/usr/bin/env bash
#
# Tests for install.sh checksum verification.
#
# Two layers:
#   1. Unit  -- sources install.sh and calls its functions directly.
#   2. E2E   -- serves a fake release over HTTP and runs a real install,
#               including the case that matters most: a corrupted archive
#               must not be installed.
#
# Usage: bash test/install/test_install.sh

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INSTALL_SH="$REPO_ROOT/install.sh"

PASS=0
FAIL=0

# Cleanup is global: RETURN traps set inside a function outlive it and re-fire
# in the caller's scope, where the temp-dir variable is no longer set.
CLEANUP_DIRS=()
SERVER_PID=""

cleanup() {
    if [[ -n "$SERVER_PID" ]]; then
        kill "$SERVER_PID" 2>/dev/null
        wait "$SERVER_PID" 2>/dev/null
    fi
    local d
    for d in "${CLEANUP_DIRS[@]+"${CLEANUP_DIRS[@]}"}"; do
        rm -rf "$d"
    done
}
trap cleanup EXIT

ok() {
    PASS=$((PASS + 1))
    echo "  PASS: $1"
}

bad() {
    FAIL=$((FAIL + 1))
    echo "  FAIL: $1" >&2
}

# assert_status <expected> <actual> <description>
assert_status() {
    if [[ "$1" == "$2" ]]; then
        ok "$3"
    else
        bad "$3 (expected exit $1, got $2)"
    fi
}

sha256_of() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | cut -d' ' -f1
    else
        shasum -a 256 "$1" | cut -d' ' -f1
    fi
}

# ---------------------------------------------------------------------------
# Unit tests
# ---------------------------------------------------------------------------

unit_tests() {
    echo "Unit tests (sourcing install.sh)"

    # Sourcing must not run the installer. The guard at the bottom of
    # install.sh is what makes this possible.
    # shellcheck source=/dev/null
    source "$INSTALL_SH"

    # install.sh sets `set -e` and an ERR trap at the top level; both leak into
    # this shell when sourced. Drop them so a function returning non-zero is a
    # test result rather than an abort.
    set +e
    trap - ERR

    local tmp
    tmp=$(mktemp -d)
    CLEANUP_DIRS+=("$tmp")

    local archive="$tmp/goingenv-v1.2.3-linux-amd64.tar.gz"
    echo "pretend archive contents" > "$archive"
    local digest
    digest=$(sha256_of "$archive")

    local status

    verify_checksum "$archive" "$digest" >/dev/null 2>&1
    assert_status 0 $? "verify_checksum accepts a matching digest"

    verify_checksum "$archive" "0000000000000000000000000000000000000000000000000000000000000000" >/dev/null 2>&1
    assert_status 1 $? "verify_checksum rejects a mismatched digest"

    # Regression guard: this used to return 0 and skip verification entirely.
    verify_checksum "$archive" "" >/dev/null 2>&1
    assert_status 1 $? "verify_checksum fails closed on an empty digest"

    # fetch_expected_checksum reads from a local file:// base, so no network.
    local rel="$tmp/releases/download/v1.2.3"
    mkdir -p "$rel"
    # shellcheck disable=SC2034  # read by fetch_expected_checksum from the sourced install.sh.
    # Deliberately not exported: the e2e section below sets it per-invocation.
    GOINGENV_DOWNLOAD_BASE="file://$tmp"

    # Happy path.
    printf '%s  goingenv-v1.2.3-linux-amd64.tar.gz\n' "$digest" > "$rel/checksums.txt"
    local got
    got=$(fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-linux-amd64.tar.gz")
    status=$?
    if [[ $status -eq 0 && "$got" == "$digest" ]]; then
        ok "fetch_expected_checksum extracts the digest for the requested archive"
    else
        bad "fetch_expected_checksum happy path (status=$status, got=$got)"
    fi

    # Regression guard: debug output must not end up inside the returned
    # digest. debug() used to write to stdout, so with DEBUG=1 the command
    # substitution captured the log lines along with the hash and every
    # verification failed with a garbled "Expected:" value.
    printf '%s  goingenv-v1.2.3-linux-amd64.tar.gz\n' "$digest" > "$rel/checksums.txt"
    local debug_got
    debug_got=$(DEBUG=1 fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-linux-amd64.tar.gz" 2>/dev/null)
    if [[ "$debug_got" == "$digest" ]]; then
        ok "fetch_expected_checksum returns a clean digest with DEBUG=1"
    else
        bad "DEBUG=1 polluted the returned digest (got: $debug_got)"
    fi

    # A different archive in the manifest must not satisfy the lookup.
    fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-darwin-arm64.tar.gz" >/dev/null 2>&1
    assert_status 1 $? "fetch_expected_checksum fails when the archive is absent from checksums.txt"

    # Suffix matching must not be possible: this entry ENDS WITH the name we ask for.
    printf '%s  x-goingenv-v1.2.3-linux-amd64.tar.gz\n' "$digest" > "$rel/checksums.txt"
    fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-linux-amd64.tar.gz" >/dev/null 2>&1
    assert_status 1 $? "fetch_expected_checksum does not suffix-match a longer filename"

    # A 200 response carrying HTML must not be read as a digest.
    printf '<!DOCTYPE html><html><body>Not Found</body></html>\n' > "$rel/checksums.txt"
    fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-linux-amd64.tar.gz" >/dev/null 2>&1
    assert_status 1 $? "fetch_expected_checksum rejects a non-hex payload"

    # Missing manifest entirely.
    rm -f "$rel/checksums.txt"
    fetch_expected_checksum "$tmp" "v1.2.3" "goingenv-v1.2.3-linux-amd64.tar.gz" >/dev/null 2>&1
    assert_status 1 $? "fetch_expected_checksum fails when checksums.txt is missing"
}

# ---------------------------------------------------------------------------
# End-to-end tests against a local fixture server
# ---------------------------------------------------------------------------

e2e_tests() {
    echo "End-to-end tests (local fixture release)"

    if ! command -v python3 >/dev/null 2>&1; then
        echo "  SKIP: python3 not available for the fixture server"
        return
    fi
    if ! command -v go >/dev/null 2>&1; then
        echo "  SKIP: go not available to build the fixture binary"
        return
    fi

    local tmp
    tmp=$(mktemp -d)
    CLEANUP_DIRS+=("$tmp")

    local version="v0.0.0-test"
    local platform
    case "$(uname -s)" in
        Linux*) platform="linux" ;;
        Darwin*) platform="darwin" ;;
        *) echo "  SKIP: unsupported OS $(uname -s)"; return ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) platform="$platform-amd64" ;;
        arm64|aarch64) platform="$platform-arm64" ;;
        *) echo "  SKIP: unsupported arch $(uname -m)"; return ;;
    esac

    local archive_name="goingenv-${version}-${platform}.tar.gz"
    local rel="$tmp/www/releases/download/$version"
    mkdir -p "$rel" "$tmp/build"

    # Build the real binary so verify_installation has something to run.
    (cd "$REPO_ROOT" && go build -o "$tmp/build/goingenv" ./cmd/goingenv) || {
        echo "  SKIP: could not build goingenv"
        return
    }
    tar -czf "$rel/$archive_name" -C "$tmp/build" goingenv

    local digest
    digest=$(sha256_of "$rel/$archive_name")
    printf '%s  %s\n' "$digest" "$archive_name" > "$rel/checksums.txt"

    # Serve the fixture.
    local port
    port=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1]);s.close()')
    (cd "$tmp/www" && python3 -m http.server "$port" --bind 127.0.0.1 >/dev/null 2>&1) &
    SERVER_PID=$!

    # Wait for the server to accept connections.
    for _ in $(seq 1 50); do
        if curl -s -o /dev/null "http://127.0.0.1:$port/" 2>/dev/null; then break; fi
        sleep 0.1
    done

    local base="http://127.0.0.1:$port"
    local status

    # 1. Happy path: matching checksum installs.
    rm -rf "${tmp:?}/bin"
    GOINGENV_DOWNLOAD_BASE="$base" bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/happy.log" 2>&1
    status=$?
    if [[ $status -eq 0 && -x "$tmp/bin/goingenv" ]]; then
        ok "install succeeds when the checksum matches"
    else
        bad "install should succeed with a matching checksum (status=$status)"
        sed 's/^/      /' "$tmp/happy.log" >&2
    fi

    # 1b. Same, with DEBUG=1: debug output on stdout would corrupt the digest.
    rm -rf "${tmp:?}/bin"
    GOINGENV_DOWNLOAD_BASE="$base" DEBUG=1 bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/debug.log" 2>&1
    status=$?
    if [[ $status -eq 0 && -x "$tmp/bin/goingenv" ]]; then
        ok "install succeeds with DEBUG=1"
    else
        bad "install should succeed with DEBUG=1 (status=$status)"
        sed 's/^/      /' "$tmp/debug.log" >&2
    fi

    # Keep a pristine copy so the corruption test can be undone.
    cp "$rel/$archive_name" "$tmp/pristine.tar.gz"

    # 2. The one that matters: corrupt the archive, leave checksums.txt alone.
    rm -rf "${tmp:?}/bin"
    printf 'corrupted' >> "$rel/$archive_name"
    GOINGENV_DOWNLOAD_BASE="$base" bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/corrupt.log" 2>&1
    status=$?
    if [[ $status -ne 0 ]]; then
        ok "install refuses a corrupted archive"
    else
        bad "install accepted a corrupted archive"
    fi
    if [[ ! -e "$tmp/bin/goingenv" ]]; then
        ok "no binary is installed when verification fails"
    else
        bad "a binary was installed despite a failed checksum"
    fi

    # Restore the good archive. The next two cases tamper with the manifest
    # instead, which isolates "the digest does not match" from "the archive is
    # unusable" -- GNU tar rejects an archive with trailing garbage while
    # bsdtar extracts it anyway, so reusing the corrupted one made the
    # --skip-checksum case pass on macOS and fail on Linux.
    cp "$tmp/pristine.tar.gz" "$rel/$archive_name"
    printf '%s  %s\n' "0000000000000000000000000000000000000000000000000000000000000000" \
        "$archive_name" > "$rel/checksums.txt"

    # 3. A good archive whose published digest does not match must still be
    #    refused: verification compares against the manifest, not the file.
    rm -rf "${tmp:?}/bin"
    GOINGENV_DOWNLOAD_BASE="$base" bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/mismatch.log" 2>&1
    status=$?
    if [[ $status -ne 0 && ! -e "$tmp/bin/goingenv" ]]; then
        ok "install refuses an archive whose published digest does not match"
    else
        bad "install accepted a digest mismatch (status=$status)"
    fi

    # 4. --skip-checksum bypasses that same mismatch and installs.
    rm -rf "${tmp:?}/bin"
    GOINGENV_DOWNLOAD_BASE="$base" bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        --skip-checksum >"$tmp/skip.log" 2>&1
    status=$?
    if [[ $status -eq 0 && -x "$tmp/bin/goingenv" ]]; then
        ok "--skip-checksum bypasses verification"
    else
        bad "--skip-checksum should have installed (status=$status)"
        sed 's/^/      /' "$tmp/skip.log" >&2
    fi

    # 5. Missing checksums.txt must fail closed.
    rm -rf "${tmp:?}/bin"
    mv "$rel/checksums.txt" "$tmp/checksums.txt.bak"
    GOINGENV_DOWNLOAD_BASE="$base" bash "$INSTALL_SH" \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/nochecksums.log" 2>&1
    status=$?
    if [[ $status -ne 0 && ! -e "$tmp/bin/goingenv" ]]; then
        ok "install fails closed when checksums.txt is unavailable"
    else
        bad "install proceeded without a checksums.txt (status=$status)"
    fi

    # Restore a correct manifest for the remaining cases.
    printf '%s  %s\n' "$digest" "$archive_name" > "$rel/checksums.txt"

    # ---------------------------------------------------------------------
    # The piped path: `curl ... | bash`. This is the ONLY way users are told
    # to install, and until this test existed nothing exercised it -- every
    # case above runs the script as a FILE. Reading from stdin leaves
    # BASH_SOURCE empty, which silently stopped main() from running at all.
    # ---------------------------------------------------------------------

    # 6. Piped --help must produce output.
    local help_out
    help_out=$(cat "$INSTALL_SH" | bash -s -- --help 2>&1)
    if [[ -n "$help_out" ]] && grep -q "USAGE" <<<"$help_out"; then
        ok "piped --help produces the usage text"
    else
        bad "piped --help produced no usage output (main did not run)"
    fi

    # 6a. ...and that usage text must be runnable. $0 is literally "bash" when
    # the script arrives on stdin, so the help used to read "USAGE: bash
    # [OPTIONS]" and offer "bash --version v1.3.0" -- copy-pasteable, and inert.
    if grep -q 'curl -fsSL .*install\.sh | bash -s -- \[OPTIONS\]' <<<"$help_out"; then
        ok "piped --help prints a pipe-aware usage line"
    else
        bad "piped --help did not print a pipe-aware usage line"
    fi

    # The load-bearing half: this assertion FAILS on the old script, which is
    # what the "produces the usage text" check above could never do.
    if ! grep -qE '(^|[[:space:]])bash (\[OPTIONS\]|--[a-z])' <<<"$help_out"; then
        ok "piped --help never presents bare 'bash' as the command"
    else
        bad "piped --help still presents bare 'bash' as the command"
    fi

    # 6b. File mode must name the script, not a curl pipeline.
    local file_help_out
    file_help_out=$(bash "$INSTALL_SH" --help 2>&1)
    if grep -qF "$INSTALL_SH [OPTIONS]" <<<"$file_help_out" \
       && ! grep -q 'curl -fsSL' <<<"$file_help_out"; then
        ok "file-mode --help names the script path"
    else
        bad "file-mode --help did not name the script path"
    fi

    # 7. A full install through the pipe.
    rm -rf "${tmp:?}/bin"
    cat "$INSTALL_SH" | GOINGENV_DOWNLOAD_BASE="$base" bash -s -- \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell --yes --force \
        >"$tmp/piped.log" 2>&1
    status=$?
    if [[ $status -eq 0 && -x "$tmp/bin/goingenv" ]]; then
        ok "piped install (curl | bash) installs the binary"
    else
        bad "piped install failed (status=$status)"
        sed 's/^/      /' "$tmp/piped.log" >&2
    fi

    # 8. Re-running the piped install upgrades in place rather than failing.
    #    This used to exit 1 in non-interactive mode, so upgrading via the
    #    documented one-liner was broken by default.
    cat "$INSTALL_SH" | GOINGENV_DOWNLOAD_BASE="$base" bash -s -- \
        --version "$version" --dir "$tmp/bin" --no-sudo --skip-shell \
        >"$tmp/rerun.log" 2>&1
    status=$?
    if [[ $status -eq 0 && -x "$tmp/bin/goingenv" ]]; then
        ok "re-running the piped install succeeds (upgrade path)"
    else
        bad "re-running the piped install failed (status=$status)"
        sed 's/^/      /' "$tmp/rerun.log" >&2
    fi

    # ---------------------------------------------------------------------
    # PATH setup. Every case above passes --skip-shell, so add_to_path had no
    # coverage at all -- which is how it shipped selecting the profile from
    # $BASH_VERSION (always set inside this bash script) and writing zsh
    # users' PATH into a bash profile they never read.
    # ---------------------------------------------------------------------

    # 9. A zsh user's PATH goes to .zshrc, not .bashrc.
    local zhome="$tmp/zhome"
    rm -rf "$zhome" "$tmp/zbin"; mkdir -p "$zhome"
    cat "$INSTALL_SH" | env HOME="$zhome" SHELL=/bin/zsh \
        GOINGENV_DOWNLOAD_BASE="$base" bash -s -- \
        --version "$version" --dir "$tmp/zbin" --no-sudo --yes --force \
        >"$tmp/zsh.log" 2>&1
    if [[ -f "$zhome/.zshrc" ]] && grep -q "$tmp/zbin" "$zhome/.zshrc"; then
        ok "zsh user gets PATH written to .zshrc"
    else
        bad "zsh user did not get PATH in .zshrc"
        # shellcheck disable=SC2012  # human-readable failure diagnostic, not parsed.
        # find -printf is GNU-only and this suite also runs on macOS.
        ls -a "$zhome" | sed 's/^/      /' >&2
    fi
    if [[ ! -f "$zhome/.bashrc" && ! -f "$zhome/.bash_profile" ]]; then
        ok "zsh user gets no stray bash profile"
    else
        bad "a bash profile was created for a zsh user"
    fi

    # 10. A bash user still gets a bash profile.
    local bhome="$tmp/bhome"
    rm -rf "$bhome" "$tmp/bbin"; mkdir -p "$bhome"
    cat "$INSTALL_SH" | env HOME="$bhome" SHELL=/bin/bash \
        GOINGENV_DOWNLOAD_BASE="$base" bash -s -- \
        --version "$version" --dir "$tmp/bbin" --no-sudo --yes --force \
        >"$tmp/bash.log" 2>&1
    if grep -qs "$tmp/bbin" "$bhome/.bashrc" "$bhome/.bash_profile"; then
        ok "bash user gets PATH written to a bash profile"
    else
        bad "bash user did not get PATH in a bash profile"
    fi
    if [[ ! -f "$zhome/.zshrc.bak" && ! -f "$bhome/.zshrc" ]]; then
        ok "bash user gets no stray .zshrc"
    else
        bad "a .zshrc was created for a bash user"
    fi

    # 11. Running PATH setup twice must not duplicate the export line.
    cat "$INSTALL_SH" | env HOME="$zhome" SHELL=/bin/zsh \
        GOINGENV_DOWNLOAD_BASE="$base" bash -s -- \
        --version "$version" --dir "$tmp/zbin" --no-sudo --yes --force \
        >"$tmp/zsh2.log" 2>&1
    local count
    count=$(grep -c "Added by goingenv installer" "$zhome/.zshrc" || true)
    if [[ "$count" -eq 1 ]]; then
        ok "re-running does not duplicate the PATH export"
    else
        bad "PATH export appears $count times in .zshrc"
    fi
}

# ---------------------------------------------------------------------------

main() {
    echo "install.sh test suite"
    echo

    bash -n "$INSTALL_SH" || {
        echo "install.sh failed syntax check" >&2
        exit 1
    }
    echo "  PASS: install.sh parses"
    PASS=$((PASS + 1))
    echo

    unit_tests
    echo
    e2e_tests
    echo

    echo "$PASS passed, $FAIL failed"
    [[ $FAIL -eq 0 ]]
}

main "$@"
