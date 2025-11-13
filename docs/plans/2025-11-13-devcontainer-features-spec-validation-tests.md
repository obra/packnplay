# Specification Validation Test Scenarios

**Purpose:** Define test cases that would verify 100% devcontainer features specification compliance

---

## Test Category 1: Feature Metadata Completeness

### Test: TS1.1 - Parse All Spec Fields
**Scenario:** Feature metadata with every specification field
**JSON:**
```json
{
  "id": "complete-feature",
  "version": "1.5.0",
  "name": "Complete Test Feature",
  "description": "Tests all specification fields",
  "options": {
    "version": {
      "type": "string",
      "default": "18",
      "description": "Component version",
      "proposals": ["16", "18", "20"]
    }
  },
  "containerEnv": {
    "FEATURE_NAME": "complete"
  },
  "remoteUser": "testuser",
  "privileged": true,
  "init": true,
  "capAdd": ["NET_ADMIN"],
  "securityOpt": ["apparmor=unconfined"],
  "entrypoint": ["/bin/custom-entry"],
  "mounts": [
    {
      "source": "test-vol",
      "target": "/test",
      "type": "volume"
    }
  ],
  "onCreateCommand": "echo 'onCreate'",
  "updateContentCommand": ["sh", "-c", "echo updateContent"],
  "postCreateCommand": "echo 'postCreate'",
  "postStartCommand": ["echo", "postStart"],
  "postAttachCommand": "echo 'postAttach'",
  "dependsOn": ["base-feature"],
  "installsAfter": ["prep-feature"]
}
```

**Verification:** All fields parsed into FeatureMetadata struct
**Status:** ✅ Would Pass (all fields exist)

---

## Test Category 2: Feature Options Processing

### Test: TS2.1 - Option Normalization
**Scenario:** Options with various naming patterns converted to ENV correctly

| Input Option | Expected ENV | Current | Status |
|--------------|-------------|---------|--------|
| `version` | `VERSION` | ✅ | ✅ |
| `install-type` | `INSTALL_TYPE` | ✅ | ✅ |
| `nodeGypDeps` | `NODEGYPDEPS` | ✅ | ✅ |
| `123test` | `_123TEST` | ✅ | ✅ |
| `test@key!` | `TEST_KEY_` | ✅ | ✅ |

**Status:** ✅ Would Pass

### Test: TS2.2 - Option Defaults Applied
**Scenario:** When user doesn't provide option, default is used

```json
{
  "options": {
    "version": {
      "type": "string",
      "default": "latest"
    }
  }
}
```

**Expected:** `VERSION=latest` in Dockerfile
**Status:** ✅ Would Pass

### Test: TS2.3 - User Option Overrides Default
**Scenario:** User-provided option overrides default

```json
{
  "options": {
    "version": {
      "type": "string",
      "default": "latest"
    }
  }
}
// User passes: {"version": "18.20.0"}
```

**Expected:** `VERSION=18.20.0` in Dockerfile
**Status:** ✅ Would Pass

### Test: TS2.4 - Option Type Validation (STRING)
**Scenario:** Reject non-string values for string-typed options

```json
{
  "options": {
    "version": {
      "type": "string",
      "default": "latest"
    }
  }
}
// User passes: {"version": 123}  // ← WRONG TYPE
```

**Expected:** Error: "option 'version' must be string, got number"
**Current:** ✅ Accepted silently
**Status:** ❌ Would Fail

### Test: TS2.5 - Option Proposals Validation
**Scenario:** Reject values not in proposals list

```json
{
  "options": {
    "version": {
      "type": "string",
      "proposals": ["16", "18", "20"]
    }
  }
}
// User passes: {"version": "14"}  // ← NOT IN PROPOSALS
```

**Expected:** Error: "option 'version'='14' not in proposals: [16, 18, 20]"
**Current:** ✅ Accepted silently
**Status:** ❌ Would Fail

### Test: TS2.6 - Number Type Options
**Scenario:** Handle numeric option values

```json
{
  "options": {
    "workers": {
      "type": "number",
      "default": 4
    }
  }
}
// User passes: {"workers": 8}
```

**Expected:** `WORKERS=8` in environment
**Current:** Parsing unclear, likely works by luck
**Status:** ⚠️ Partial

### Test: TS2.7 - Boolean Type Options
**Scenario:** Handle boolean option values

```json
{
  "options": {
    "installTools": {
      "type": "boolean",
      "default": false
    }
  }
}
// User passes: {"installTools": true}
```

**Expected:** `INSTALLTOOLS=true` in environment
**Status:** ⚠️ Partial

---

## Test Category 3: Feature Container Properties

### Test: TS3.1 - Privileged Mode
**Scenario:** Feature requests privileged container

```json
{
  "privileged": true
}
```

**Expected:** Docker run args include `--privileged`
**Verification:** `docker inspect <container> | grep Privileged`
**Status:** ✅ Code exists, but E2E test missing

### Test: TS3.2 - Capabilities (capAdd)
**Scenario:** Feature adds Linux capabilities

```json
{
  "capAdd": ["NET_ADMIN", "SYS_PTRACE"]
}
```

**Expected:** Docker run args include `--cap-add NET_ADMIN --cap-add SYS_PTRACE`
**Verification:** `docker inspect <container> | grep CapAdd`
**Status:** ✅ Code exists, but E2E test missing

### Test: TS3.3 - Security Options
**Scenario:** Feature sets security options

```json
{
  "securityOpt": ["apparmor=unconfined"]
}
```

**Expected:** Docker run args include `--security-opt apparmor=unconfined`
**Status:** ✅ Code exists, but E2E test missing

### Test: TS3.4 - Init Process
**Scenario:** Feature enables init

```json
{
  "init": true
}
```

**Expected:** Docker run args include `--init`
**Status:** ❌ Code missing

### Test: TS3.5 - Entrypoint Override
**Scenario:** Feature overrides container entrypoint

```json
{
  "entrypoint": ["/usr/bin/dumb-init", "--"]
}
```

**Expected:** Docker run args include `--entrypoint /usr/bin/dumb-init` `--`
**Status:** ❌ Code missing

### Test: TS3.6 - Container Environment (containerEnv)
**Scenario:** Feature sets environment variables

```json
{
  "containerEnv": {
    "FEATURE_VAR": "value",
    "RUST_BACKTRACE": "1"
  }
}
```

**Expected:** 
- Dockerfile includes `ENV FEATURE_VAR=value`
- Container has these variables set
**Status:** ✅ Works via Dockerfile ENV

---

## Test Category 4: Feature Mounts

### Test: TS4.1 - Volume Mount
**Scenario:** Feature requests volume mount

```json
{
  "mounts": [
    {
      "source": "cache-volume",
      "target": "/cache",
      "type": "volume"
    }
  ]
}
```

**Expected:** 
- Docker run includes `--mount type=volume,source=cache-volume,target=/cache`
- Volume visible in container at `/cache`
**Status:** ❌ Code missing

### Test: TS4.2 - Bind Mount
**Scenario:** Feature requests bind mount

```json
{
  "mounts": [
    {
      "source": "/var/run/docker.sock",
      "target": "/var/run/docker.sock",
      "type": "bind"
    }
  ]
}
```

**Expected:**
- Docker run includes `--mount type=bind,source=/var/run/docker.sock,target=/var/run/docker.sock`
- Docker socket accessible in container
**Status:** ❌ Code missing

### Test: TS4.3 - Multiple Mounts
**Scenario:** Feature requests multiple mounts

```json
{
  "mounts": [
    {"source": "vol1", "target": "/data1", "type": "volume"},
    {"source": "vol2", "target": "/data2", "type": "volume"}
  ]
}
```

**Expected:** Both mounts applied
**Status:** ❌ Code missing

---

## Test Category 5: Lifecycle Commands

### Test: TS5.1 - Feature onCreate Before User onCreate
**Scenario:** Both feature and user have onCreate commands

**Feature metadata:**
```json
{
  "onCreateCommand": "echo 'feature-create' > /tmp/order.log"
}
```

**User config:**
```json
{
  "onCreateCommand": "echo 'user-create' >> /tmp/order.log"
}
```

**Expected:** /tmp/order.log contains:
```
feature-create
user-create
```

**Verification:** Feature command runs first, user second
**Status:** ✅ Code exists, test exists

### Test: TS5.2 - Feature postCreate Before User postCreate
**Scenario:** Both have postCreate commands
**Expected:** Feature runs first
**Status:** ✅ Code exists, test exists

### Test: TS5.3 - Feature postStart Before User postStart
**Scenario:** Both have postStart commands
**Expected:** Feature runs first
**Status:** ⚠️ Code exists, no E2E test

### Test: TS5.4 - updateContentCommand Execution
**Scenario:** Feature and/or user have updateContentCommand

**Note:** packnplay doesn't have content-sync concept
**Possible implementation:** Execute on first container start as approximation

**Current:** Not in Config struct
**Status:** ❌ Missing

### Test: TS5.5 - postAttachCommand Execution
**Scenario:** Feature and/or user have postAttachCommand

**Note:** Only relevant for IDE extensions (VS Code)
**Possible implementation:** Skip with note that it's IDE-specific

**Current:** Not executed
**Status:** ❌ Missing

### Test: TS5.6 - Mixed Command Formats
**Scenario:** Feature has different command formats

**Feature metadata:**
```json
{
  "onCreateCommand": "echo test",                    // string
  "postCreateCommand": ["npm", "install"],           // array
  "postStartCommand": {                              // object (parallel)
    "server": "npm start",
    "watch": "npm run watch"
  }
}
```

**Expected:** All formats handled correctly
**Status:** ⚠️ Parsing works, execution partial

---

## Test Category 6: Feature Dependencies

### Test: TS6.1 - Linear Dependency Chain
**Scenario:** Feature A depends on B depends on C

```json
{
  "features": {
    "feature-a": {},  // depends on B
    "feature-b": {},  // depends on C
    "feature-c": {}   // no dependencies
  }
}
```

**Expected:** Installation order: C, B, A
**Status:** ✅ Test exists, passes

### Test: TS6.2 - Diamond Dependency
**Scenario:**
```
      A
     / \
    B   C
     \ /
      D
```

```json
{
  "features": {
    "d": {},
    "b": {"dependsOn": ["d"]},
    "c": {"dependsOn": ["d"]},
    "a": {"dependsOn": ["b", "c"]}
  }
}
```

**Expected:** Installation order: D, B, C, A (or D, C, B, A)
**Status:** ❌ Not tested

### Test: TS6.3 - Soft Dependency (installsAfter)
**Scenario:** Feature B should install after A, but doesn't strictly depend

```json
{
  "features": {
    "b": {"installsAfter": ["a"]},
    "a": {}
  }
}
```

**Expected:** B installs after A if A exists, otherwise B installs first
**Status:** ⚠️ Logic exists, not fully tested

### Test: TS6.4 - Circular Dependency Detection
**Scenario:** Feature A depends on B, B depends on A

```json
{
  "features": {
    "a": {"dependsOn": ["b"]},
    "b": {"dependsOn": ["a"]}
  }
}
```

**Expected:** Error: "Circular dependency detected"
**Status:** ❌ Would hang/infinite loop

### Test: TS6.5 - Missing Dependency
**Scenario:** Feature A depends on non-existent feature B

```json
{
  "features": {
    "a": {"dependsOn": ["missing-b"]}
  }
}
```

**Expected:** Error: "Feature 'a' depends on missing feature 'missing-b'"
**Status:** ❌ Current: "cannot resolve dependencies: features [a] have unsatisfied dependencies"

---

## Test Category 7: Real-World Features

### Test: TS7.1 - Docker-in-Docker Feature
**Feature:** `ghcr.io/devcontainers/features/docker-in-docker:2`

**Options:**
```json
{
  "version": "latest",
  "enableNonRootDocker": true
}
```

**Verification:**
- ✅ Feature installs
- ✅ Options processed
- ❌ Privileged mode applied
- ❌ Mounts applied (docker socket)
- ✅ Lifecycle commands run
- **Result:** ❌ docker-in-docker won't work without privileged+mounts

### Test: TS7.2 - Node Feature with Version
**Feature:** `ghcr.io/devcontainers/features/node:1`

**Options:**
```json
{
  "version": "18.20.0"
}
```

**Verification:**
- ✅ Feature installs (tested)
- ✅ Options processed (tested)
- ✅ Version applied
- **Result:** ✅ Works correctly

### Test: TS7.3 - Go Feature
**Feature:** `ghcr.io/devcontainers/features/go:1`

**Options:**
```json
{
  "version": "1.20.3",
  "nodeVersion": "18"
}
```

**Verification:**
- Options processed
- Correct versions installed
- **Result:** ❌ Not tested

### Test: TS7.4 - Python Feature
**Feature:** `ghcr.io/devcontainers/features/python:1`

**Options:**
```json
{
  "version": "3.11.2",
  "installTools": true
}
```

**Verification:**
- Options processed
- Python and tools installed
- **Result:** ❌ Not tested

### Test: TS7.5 - PostgreSQL Feature
**Feature:** `ghcr.io/devcontainers/features/postgres:1`

**Verification:**
- Lifecycle commands setup database
- Features with mounts work
- **Result:** ⚠️ Needs mount support

---

## Test Category 8: Error Handling

### Test: TS8.1 - Invalid Option Type
**Scenario:** User provides wrong type for option

```json
{
  "options": {
    "version": {"type": "number", "default": 1}
  }
}
// User: {"version": "not-a-number"}
```

**Expected:** Clear error message
**Status:** ❌ Silent acceptance

### Test: TS8.2 - OCI Pull Failure
**Scenario:** Feature fails to pull from registry

```json
{
  "features": {
    "ghcr.io/nonexistent/feature:1": {}
  }
}
```

**Expected:** Clear error message
**Current:** Likely Docker/oras error
**Status:** ⚠️ Partial

### Test: TS8.3 - Invalid Feature Metadata
**Scenario:** Feature metadata JSON is invalid

**Expected:** Parse error with helpful message
**Status:** ⚠️ Works but error message quality unknown

### Test: TS8.4 - Missing Required Feature Fields
**Scenario:** Feature missing 'id' or 'version'

**Expected:** Error identifying missing field
**Status:** ⚠️ Likely accepts invalid metadata

### Test: TS8.5 - Feature Install Script Fails
**Scenario:** Feature's install.sh exits with error code

**Expected:** Clear error message
**Status:** ❌ Behavior unknown

---

## Summary: Test Coverage

| Category | Total Tests | Passing | Missing | Status |
|----------|-------------|---------|---------|--------|
| Metadata | 1 | 1 | 0 | ✅ |
| Options | 7 | 3 | 4 | ❌ |
| Properties | 6 | 2 | 4 | ⚠️ |
| Mounts | 3 | 0 | 3 | ❌ |
| Lifecycle | 6 | 2 | 4 | ⚠️ |
| Dependencies | 5 | 1 | 4 | ❌ |
| Real-World | 5 | 1 | 4 | ❌ |
| Error Handling | 5 | 0 | 5 | ❌ |

**Total:** 38 test scenarios
**Passing:** 10 (~26%)
**Missing:** 28 (~74%)

---

## Priority for Implementation

### CRITICAL (Block compliance)
1. Feature mounts (TS4.1-4.3)
2. Option validation (TS2.4-2.7)
3. Init/Entrypoint (TS3.4-3.5)
4. Missing lifecycle hooks (TS5.4-5.5)

### HIGH (Important functionality)
5. Docker-in-Docker E2E (TS7.1)
6. Error handling (TS8.1-8.5)
7. Diamond dependencies (TS6.2)
8. Circular dependency detection (TS6.4)

### MEDIUM (Completeness)
9. More real-world feature testing (TS7.3-7.5)
10. Mixed command formats (TS5.6)

