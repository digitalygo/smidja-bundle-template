import argparse
from decimal import Decimal
import hashlib
import io
import json
from pathlib import Path, PurePosixPath
import re
import stat
from urllib.parse import urlsplit
import zipfile

TARGETS = ("linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64")
GROUPS = ("teams", "workflows", "tools", "skills")
FIELDS = {"schema_version", "id", "version", "publisher", "requires_smidja", "scopes", "targets", *GROUPS, "files", "dependencies", "configuration"}
MAX_BYTES = 64 << 20


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encoded(value):
    return (json.dumps(value, indent=2, sort_keys=True) + "\n").encode()


def relative(name):
    require(isinstance(name, str) and name and name != "." and "\\" not in name and ":" not in name and "\0" not in name, "invalid payload path")
    path = PurePosixPath(name)
    require(not path.is_absolute() and ".." not in path.parts and str(path) == name, "invalid payload path")
    return name


def https_url(value):
    require(isinstance(value, str) and value and not any(character.isspace() or ord(character) < 32 or ord(character) == 127 for character in value), "source URL must not contain whitespace or controls")
    require(not any(character in value for character in "?#\\") and re.search(r"%(?![0-9a-fA-F]{2})", value) is None, "source URL has invalid delimiters or escaping")
    url = urlsplit(value)
    require(url.scheme == "https" and url.hostname and url.username is None and url.password is None, "source requires clean HTTPS URL")
    require(re.fullmatch(r"(?:\[[0-9a-fA-F:.]+\]|[^:/@\[\]\\%]+)(?::[0-9]+)?", url.netloc), "source URL has invalid authority")
    require(url.port is None or 1 <= url.port <= 65535, "source URL has invalid port")
    return value.rstrip("/")


class SchemaNumber(Decimal):
    def __new__(cls, token):
        value = super().__new__(cls, token)
        value.token = token
        return value


def compact_schema(value):
    if type(value) in {int, float, Decimal, SchemaNumber}:
        require(bounded_number(value), "schema numbers exceed supported 4096-character/exponent domain")
        return getattr(value, "token", str(value))
    if isinstance(value, dict):
        return "{" + ",".join(json.dumps(key, ensure_ascii=False) + ":" + compact_schema(item) for key, item in value.items()) + "}"
    if isinstance(value, list):
        return "[" + ",".join(compact_schema(item) for item in value) + "]"
    return json.dumps(value, ensure_ascii=False, separators=(",", ":"), allow_nan=False)


def bounded_number(value):
    if type(value) not in {int, float, Decimal, SchemaNumber}:
        return False
    token = getattr(value, "token", str(value))
    exponent = re.search(r"[eE]([+-]?[0-9]+)$", token)
    return len(token) <= 4096 and (not exponent or -4096 <= int(exponent[1]) <= 4096) and Decimal(token).is_finite()


def same_json(left, right):
    if type(left) is bool or type(right) is bool:
        return type(left) is type(right) and left == right
    if isinstance(left, (int, float, Decimal)) and isinstance(right, (int, float, Decimal)):
        return left == right
    if type(left) is not type(right):
        return False
    if isinstance(left, dict):
        return left.keys() == right.keys() and all(same_json(value, right[key]) for key, value in left.items())
    if isinstance(left, list):
        return len(left) == len(right) and all(same_json(a, b) for a, b in zip(left, right))
    return left == right


def validate_input_schema(schema, depth=0):
    require(isinstance(schema, dict) and depth <= 32, "input_schema must contain objects at depth at most 32")
    if depth == 0:
        require(len(compact_schema(schema).encode()) <= 64 << 10, "input_schema exceeds 64 KiB")
    for key, value in schema.items():
        if key in {"title", "description"}:
            require(isinstance(value, str), key + " must be a string")
        elif key == "type":
            require(isinstance(value, str) and value in {"object", "array", "string", "number", "integer", "boolean", "null"}, "unsupported schema type")
        elif key == "properties":
            require(isinstance(value, dict), "properties must be an object")
            for child in value.values():
                validate_input_schema(child, depth + 1)
        elif key == "items":
            validate_input_schema(value, depth + 1)
        elif key == "required":
            require(isinstance(value, list) and all(isinstance(name, str) for name in value) and len(set(value)) == len(value), "required must contain unique strings")
        elif key == "additionalProperties":
            require(type(value) is bool, "additionalProperties must be boolean")
        elif key == "enum":
            require(isinstance(value, list) and value, "enum must be a nonempty array")
            for index, item in enumerate(value):
                require(not any(same_json(item, previous) for previous in value[:index]), "enum values must be unique")
        elif key == "const":
            pass
        elif key in {"minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems"}:
            require(bounded_number(value), key + " must be a bounded numeric value")
            if key not in {"minimum", "maximum"}:
                require(value >= 0 and value <= (1 << 63) - 1 and int(value) == value, key + " must be a nonnegative bounded integer")
        else:
            raise ValueError("unsupported input_schema keyword " + key)


def validate_workflow_contract(document):
    if "input_schema" in document:
        validate_input_schema(document["input_schema"])
    if "allowed_invokers" not in document or document["allowed_invokers"] is None:
        return
    invokers = document["allowed_invokers"]
    require(isinstance(invokers, list), "allowed_invokers must be an array")
    seen = set()
    for invoker in invokers:
        require(isinstance(invoker, str) and (invoker == "ceo" or invoker.startswith("team:") and invoker[5:].strip()) and invoker not in seen, "invalid or duplicate allowed_invoker")
        seen.add(invoker)


def validate(manifest, files):
    require(set(manifest) <= FIELDS, "unknown manifest field")
    require(manifest.get("schema_version") == 1, "schema_version must be 1")
    package_id = manifest.get("id", "")
    require(re.fullmatch(r"[a-z0-9][a-z0-9.-]{0,127}", package_id), "invalid package ID")
    version = r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)"
    require(re.fullmatch(version, manifest.get("version", "")), "version must be x.y.z")
    require(re.fullmatch(">=" + version, manifest.get("requires_smidja", "")), "requires_smidja must be >=x.y.z")
    require(str(manifest.get("publisher", "")).strip(), "publisher required")
    require(manifest.get("targets") and set(manifest["targets"]) <= set(TARGETS), "unsupported target")
    require(manifest.get("scopes") and set(manifest["scopes"]) <= {"manager", "client"}, "unsupported scope")
    require(len(files) <= 4095 and sum(map(len, files.values())) <= MAX_BYTES, "payload too large")
    for name in files:
        relative(name)
        require(name != "smidja.json", "manifest cannot inventory itself")
    references = set()
    documents = {}
    for group in GROUPS:
        for name in manifest.get(group, []):
            require(name in files and name not in references, "missing or repeated catalog file")
            references.add(name)
            if group == "skills":
                files[name].decode("utf-8")
                continue
            document = json.loads(files[name], parse_float=SchemaNumber, parse_int=SchemaNumber) if group == "workflows" else json.loads(files[name])
            identity = document.get("id", "")
            require(identity.startswith(package_id + "/") and len(identity) > len(package_id) + 1 and identity not in documents, "catalog ID must be unique and namespaced")
            documents[identity] = (group, document)
    for identity, (group, document) in documents.items():
        if group == "teams":
            require(isinstance(document.get("name"), str) and document["name"].strip() and isinstance(document.get("agents"), list) and document["agents"], "team needs name and agents")
            agent_names = set()
            for agent in document["agents"]:
                require(isinstance(agent, dict) and all(isinstance(agent.get(field), str) and agent[field].strip() for field in ("name", "model", "instructions")), "agent needs nonempty name, model and instructions")
                require(len(agent["name"].encode("utf-8")) <= 255 and agent["name"] not in agent_names, "agent names must be unique and at most 255 bytes")
                agent_names.add(agent["name"])
                for tool in agent.get("tools", []):
                    require(tool in documents and documents[tool][0] == "tools" or tool in {"bash", "read", "write", "edit", "grep", "list", "git_status", "git_diff"}, "unknown team tool")
        if group == "tools":
            require(document.get("name") and isinstance(document.get("parameters"), dict) and document.get("command"), "tool definition incomplete")
            require(document.get("client_id", "") in {"", "manager"}, "tools bind to installation")
            timeout = document.get("timeout_seconds", 0)
            require(type(timeout) is int and 0 <= timeout <= 600, "timeout_seconds must be 0..600")
            for argument in document["command"]:
                if argument.startswith("./"):
                    require(relative(argument[2:]) in files, "command file missing")
            for key, value in document.get("environment", {}).items():
                require(re.fullmatch(r"[A-Z_][A-Z0-9_]*", key) and re.fullmatch(r"[A-Z_][A-Z0-9_]*", value), "environment values must be references")
        if group == "workflows":
            validate_workflow_contract(document)
            phases = document.get("phases", [])
            require(phases, "workflow phases required")
            seen = set()
            for phase in phases:
                require(phase.get("id") and phase["id"] not in seen and phase.get("name"), "invalid phase identity")
                seen.add(phase["id"])
                require(phase.get("team_ref") in documents and documents[phase["team_ref"]][0] == "teams", "unknown workflow team")
    for name, configuration in manifest.get("configuration", {}).items():
        require(re.fullmatch(r"[A-Z_][A-Z0-9_]*", name) and set(configuration) <= {"secret"} and isinstance(configuration.get("secret"), bool), "invalid configuration")
    dependencies = set()
    for dependency in manifest.get("dependencies", []):
        require(set(dependency) == {"id", "version", "digest"} and re.fullmatch(r"[a-z0-9][a-z0-9.-]{0,127}", dependency["id"]), "invalid dependency")
        require(dependency["id"] != package_id and dependency["id"] not in dependencies and re.fullmatch(version, dependency["version"]) and re.fullmatch(r"[a-f0-9]{64}", dependency["digest"]), "invalid dependency pin")
        dependencies.add(dependency["id"])


def archive(source, target):
    source = Path(source)
    manifest = json.loads((source / "smidja.json").read_text())
    require(target in TARGETS and target in manifest.get("targets", []), "unsupported target")
    files = {}
    executable = set()
    for path in sorted(source.rglob("*")):
        require(not path.is_symlink(), "symlinks are not package payloads")
        if path.is_dir():
            continue
        require(path.is_file(), "payload must be regular files")
        name = path.relative_to(source).as_posix()
        if name == "smidja.json":
            continue
        files[name] = path.read_bytes()
        if path.stat().st_mode & 0o111:
            executable.add(name)
    manifest["targets"] = [target]
    validate(manifest, files)
    manifest["files"] = [{"path": name, "sha256": digest(data), **({"executable": True} if name in executable else {})} for name, data in sorted(files.items())]
    files["smidja.json"] = encoded(manifest)
    output = io.BytesIO()
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED) as bundle:
        for name, data in sorted(files.items()):
            entry = zipfile.ZipInfo(name, (1980, 1, 1, 0, 0, 0))
            entry.create_system = 3
            entry.external_attr = (stat.S_IFREG | (0o755 if name in executable else 0o644)) << 16
            entry.compress_type = zipfile.ZIP_DEFLATED
            bundle.writestr(entry, data)
    result = output.getvalue()
    require(len(result) <= MAX_BYTES, "archive too large")
    validate_archive(result)
    return manifest, result


def validate_archive(data):
    require(len(data) <= MAX_BYTES, "archive too large")
    with zipfile.ZipFile(io.BytesIO(data)) as bundle:
        require(len(bundle.infolist()) <= 4096 and sum(entry.file_size for entry in bundle.infolist()) <= MAX_BYTES, "archive too large")
        files = {}
        for entry in bundle.infolist():
            relative(entry.filename)
            require(entry.filename not in files and stat.S_ISREG(entry.external_attr >> 16), "duplicate or nonregular entry")
            files[entry.filename] = bundle.read(entry)
    manifest = json.loads(files.pop("smidja.json"))
    validate(manifest, files)
    inventory = manifest["files"]
    require(len(inventory) == len(files) and len({entry["path"] for entry in inventory}) == len(files), "inventory mismatch")
    for entry in inventory:
        require(entry["path"] in files and digest(files[entry["path"]]) == entry["sha256"], "payload digest mismatch")
    return manifest


def build(source, output, target, archive_base_url):
    base = https_url(archive_base_url)
    manifest, data = archive(source, target)
    directory = Path(output) / target.replace("/", "-")
    directory.mkdir(parents=True, exist_ok=True)
    (directory / "package.zip").write_bytes(data)
    release = {"id": manifest["id"], "version": manifest["version"], "archive_url": base + "/" + target.replace("/", "-") + "/package.zip", "sha256": digest(data)}
    (directory / "release.json").write_bytes(encoded(release))
    return directory


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("source")
    parser.add_argument("output")
    parser.add_argument("target", choices=TARGETS)
    parser.add_argument("archive_base_url")
    args = parser.parse_args()
    print(build(args.source, args.output, args.target, args.archive_base_url))


if __name__ == "__main__":
    main()
