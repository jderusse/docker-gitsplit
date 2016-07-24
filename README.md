# A Docker image with git and splitsh-lite

See [the official site](https://github.com/splitsh/lite) for more information about splitsh.

# Usage

Include a `.gitsplit.yml` file in the root of your repository.
This section provides a brief overview of the configuration file and split process.

Example .gitsplit.yml configuration:

```yaml
# Used to speed up the split over time by reusing git's objects
cache_dir: "/var/lib/gitsplit/my_project"

# Path to the repository to split (default = current path)
project_dir: /home/me/workspace/another_project

# List of splits.
splits:
  - prefix: "src/partA"
    target: "git@github.com/my_company/project-partA.git"
  - prefix: "src/partB"
    target:
      # You can push the split to several repositories
      - "git@github.com/my_company/project-partB.git"
      - "git@github.com/my_company/project-partZ.git"
  - prefix:
      # You can use several prefix in the split
      - "src/subTree/PartC"
      - "src/subTree/PartZ"
    target: "git@github.com/my_company/project-partC.git"

# List of references to split (defined as regexp)
origins:
  - ^master$
  - ^develop$
  - ^feature/
```

# Split your repo manualy

```
$ docker run --rm -ti -v $PWD:/srv jderusse/gitsplit
```

# Sample with drone.io (you have to handle remote repository authentication)

Beware, the container have to push on your splited repository.
It could be a security issue. Use environments variables as defined in the official documentation

```yaml
cache_dir: "/cache/gitsplit"
splits:
  "src/partA": "git@github.com/my_company/project-partA.git"
  "src/partB": "git@github.com/my_company/project-partB.git"
  "src/subTree/PartC": "git@github.com/my_company/project-partC.git"

origins:
  - ^master$
  - ^develop$
```

```yaml
# .drone.yml
pipeline:
  split:
    image: jderusse/gitsplit
    pull: true
    volumes:
      # Share a cache mounted in the runner
      - /drone/cache/gitsplit:/cache/gitsplit

      # Use ssh key defined in the runner
      - /drone/env/gitsplit.ssh:/root/.ssh/
    commands:
      # have to fetch remote branches
      - git fetch
      - gitsplit
```
