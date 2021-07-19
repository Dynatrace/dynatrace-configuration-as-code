### Creating New Version

1. First, make sure your content in the `documentation/docs` directory is ready to be frozen as a version. A version always should be based from master.

- Copy the full `documentation/docs/` folder contents from above into a new `documentation/versioned_docs/version-<version>/` folder.
- Create a versioned sidebars file in `documentation/versioned_sidebars/version-<version>-sidebars.json`.
- Append the new version number to `documentation/versions.json`.

## Docs {#docs}

### Creating new docs {#creating-new-docs}

1. Place the new file into the corresponding version folder.
1. Include the reference for the new file into the corresponding sidebar file, according to version number.

**Next version docs**

```shell
# The new file.
documenatation/docs/new.md
```

**Older version docs**

```shell
# The new file.
documenatation/versioned_docs/version-1.0.0/new.md

# Edit the corresponding sidebar file.
documenatation/versioned_sidebars/version-1.0.0-sidebars.json
```

## Versions {#versions}

Each directory in `documentation/versioned_docs/` will represent a documentation version.

### Updating an existing version {#updating-an-existing-version}

You can update multiple docs versions at the same time because each directory in `documentation/versioned_docs/` represents specific routes when published.

1. Edit any file.
1. Commit and push changes.
1. It will be published to the version.

Example: When you change any file in `documentation/versioned_docs/version-2.6/`, it will only affect the docs for version `2.6`.

### Deleting an existing version {#deleting-an-existing-version}

You can delete/remove versions as well.

1. Remove the version from `documentation/versions.json`.

Example:

```diff {4}
[
  "2.0.0",
  "1.9.0",
- "1.8.0"
]
```

2. Delete the versioned docs directory. Example: `documentation/versioned_docs/version-1.8.0`.
3. Delete the versioned sidebars file. Example: `documentation/versioned_sidebars/version-1.8.0-sidebars.json`.