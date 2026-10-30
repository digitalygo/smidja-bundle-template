import copy
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import zipfile
import package_builder as builder


class PackageTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.source = self.root / "source"
        self.source.mkdir()
        self.manifest = json.loads((Path(__file__).resolve().parents[1] / "package/smidja.json").read_text())
        self.payload = {}
        original = Path(__file__).resolve().parents[1] / "package"
        for path in original.rglob("*"):
            if path.is_file() and path.name != "smidja.json":
                self.payload[path.relative_to(original).as_posix()] = path.read_bytes()
        self.payload["bin/helper"] = b"native-test-fixture"
        self.write_source()

    def write_source(self):
        (self.source / "smidja.json").write_bytes(builder.encoded(self.manifest))
        for name, data in self.payload.items():
            path = self.source / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        (self.source / "bin/helper").chmod(0o755)

    def test_build_all_targets_and_determinism(self):
        for target in builder.TARGETS:
            output = builder.build(self.source, self.root / "dist", target, "https://example.com/1.0.0/")
            data = (output / "package.zip").read_bytes()
            manifest = builder.validate_archive(data)
            self.assertEqual(manifest["targets"], [target])
            self.assertTrue(next(entry for entry in manifest["files"] if entry["path"] == "bin/helper")["executable"])
            release = json.loads((output / "release.json").read_text())
            self.assertEqual(release["sha256"], builder.digest(data))
            self.assertEqual(release["archive_url"], "https://example.com/1.0.0/" + target.replace("/", "-") + "/package.zip")
            self.assertEqual(builder.archive(self.source, target)[1], data)

    def test_manifest_rejections(self):
        variants = [
            {"schema_version": 0}, {"id": "../bad"}, {"version": "v1.0.0"},
            {"requires_smidja": "^1.0.0"}, {"publisher": ""},
            {"targets": ["linux/armv7"]}, {"scopes": ["remote"]},
            {"unknown": True}, {"teams": ["missing.json"]},
            {"skills": ["teams/reviewer.json"]},
            {"configuration": {"bad-name": {"secret": True}}},
            {"configuration": {"TOKEN": {"secret": "yes"}}},
            {"dependencies": [{"id": "other", "version": "1.0.0", "digest": "bad"}]},
            {"dependencies": [{"id": "example.coding", "version": "1.0.0", "digest": "a" * 64}]},
        ]
        for variant in variants:
            with self.subTest(variant=variant), self.assertRaises(ValueError):
                builder.validate(self.manifest | variant, self.payload)
        builder.validate(self.manifest | {"dependencies": [{"id": "other", "version": "1.0.0", "digest": "a" * 64}], "configuration": {"TOKEN": {"secret": True}}}, self.payload)

    def test_catalog_rejections(self):
        changes = [
            ("teams/reviewer.json", {"id": "foreign/reviewer"}),
            ("teams/reviewer.json", {"agents": []}),
            ("teams/reviewer.json", {"agents": [{"name": "a"}]}),
            ("teams/reviewer.json", {"agents": [{"name": "a", "instructions": "x", "tools": ["missing"]}]}),
            ("tools/echo.json", {"id": "example.coding/reviewer"}),
            ("tools/echo.json", {"parameters": []}),
            ("tools/echo.json", {"command": ["./missing"]}),
            ("tools/echo.json", {"client_id": "laptop"}),
            ("tools/echo.json", {"timeout_seconds": -1}),
            ("tools/echo.json", {"timeout_seconds": 601}),
            ("tools/echo.json", {"timeout_seconds": True}),
            ("tools/echo.json", {"environment": {"KEY": "literal secret"}}),
            ("workflows/pr-review.json", {"phases": []}),
            ("workflows/pr-review.json", {"phases": [{"id": "a", "name": "a", "team_ref": "missing"}]}),
            ("workflows/pr-review.json", {"phases": [{"id": "", "team_ref": "example.coding/reviewer"}]}),
        ]
        for name, change in changes:
            payload = dict(self.payload)
            payload[name] = builder.encoded(json.loads(payload[name]) | change)
            with self.subTest(change=change), self.assertRaises(ValueError):
                builder.validate(self.manifest, payload)
        payload = dict(self.payload)
        document = json.loads(payload["tools/echo.json"])
        document["environment"] = {"KEY": "API_KEY"}
        payload["tools/echo.json"] = builder.encoded(document)
        builder.validate(self.manifest, payload)

    def test_paths_urls_and_symlinks(self):
        for name in ["", ".", "../a", "/a", "a/../b", "a//b", "a\\b", "a:b", "a\0b"]:
            with self.subTest(name=name), self.assertRaises(ValueError):
                builder.relative(name)
        for url in ["http://example.com/a", "https://user:pass@example.com/a", "https://example.com/a?token=x", "https://example.com/#x", "/local"]:
            with self.subTest(url=url), self.assertRaises(ValueError):
                builder.https_url(url)
        with self.assertRaises(ValueError):
            builder.archive(self.source, "linux/armv7")
        (self.source / "link").symlink_to(self.source / "bin/helper")
        with self.assertRaises(ValueError):
            builder.archive(self.source, "linux/amd64")

    def test_archive_tampering(self):
        _, original = builder.archive(self.source, "linux/amd64")
        for mutation in ["digest", "missing", "unlisted", "duplicate", "directory", "traversal", "inventory"]:
            source = zipfile.ZipFile(io.BytesIO(original))
            output = io.BytesIO()
            with zipfile.ZipFile(output, "w") as target:
                for entry in source.infolist():
                    data = source.read(entry)
                    if mutation == "missing" and entry.filename == "bin/helper":
                        continue
                    if mutation == "digest" and entry.filename == "bin/helper":
                        data += b"x"
                    if mutation == "inventory" and entry.filename == "smidja.json":
                        manifest = json.loads(data)
                        manifest["files"].append(copy.deepcopy(manifest["files"][0]))
                        data = builder.encoded(manifest)
                    target.writestr(entry, data)
                if mutation in {"unlisted", "traversal", "directory"}:
                    target.writestr({"unlisted": "extra", "traversal": "../evil", "directory": "folder/"}[mutation], "x")
                if mutation == "duplicate":
                    target.writestr(source.getinfo("bin/helper"), b"again")
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                builder.validate_archive(output.getvalue())

    def test_size_limits_and_cli(self):
        with patch.object(builder, "MAX_BYTES", 2), self.assertRaises(ValueError):
            builder.archive(self.source, "linux/amd64")
        _, data = builder.archive(self.source, "linux/amd64")
        with patch.object(builder, "MAX_BYTES", 2), self.assertRaises(ValueError):
            builder.validate_archive(data)
        with patch("sys.argv", ["package_builder", str(self.source), str(self.root / "cli"), "linux/amd64", "https://example.com/pkg"]):
            builder.main()
        self.assertTrue((self.root / "cli/linux-amd64/release.json").exists())


if __name__ == "__main__":
    unittest.main()
