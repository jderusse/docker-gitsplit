# A Docker image with git and splitsh-lite

See [the official site](https://github.com/splitsh/lite) for more information about splitsh.

Demo available here [jderusse/test-split-a](https://github.com/jderusse/test-split-a).

# Usage

Include a `.gitsplit.yml` file in the root of your repository.
This section provides a brief overview of the configuration file and split process.

Use env variable to inject your credential and manage authentication.

Example `.gitsplit.yml` configuration:

```yaml
# Path to a cache directory Used to speed up the split over time by reusing git's objects
cache_url: "/cache/gitsplit"
# cache_url: "file:///cache/gitsplit"
# cache_url: "https://${GH_TOKEN}@github.com/my_company/project-cache.git"
# cache_url: "git@gitlab.com:my_company/project-cache.git"

# Path to the repository to split (default = current path)
# project_url: /home/me/workspace/another_project
# project_url: ~/workspace/another_project
# project_url: ../another_project
# project_url: file://~/workspace/another_project
# project_url: "https://${GH_TOKEN}@github.com/my_company/project.git"
# project_url: "git@gitlab.com:my_company/project.git"

# List of splits.
splits:
  - prefix: "src/partA"
    target: "https://${GH_TOKEN}@github.com/my_company/project-partA.git"
  - prefix: "src/partB"
    target:
      # You can push the split to several repositories
      - "https://${GH_TOKEN}@github.com/my_company/project-partB.git"
      - "https://${GH_TOKEN}@github.com/my_company/project-partZ.git"
  - prefix:
      # You can use several prefix in the split
      - "src/subTree/PartC:"
      - "src/subTree/PartZ:lib/z"
    target: "https://${GH_TOKEN}@github.com/my_company/project-partC.git"

# List of references to split (defined as regexp)
origins:
  - ^master$
  - ^develop$
  - ^feature/
  - ^v\d+\.\d+\.\d+$
```

# Split your repo manualy

With a github token:
```
$ docker run --rm -ti -e GH_TOKEN -v /cache:/cache/gitsplit -v $PWD:/srv jderusse/gitsplit
```

With ssh agent:
```
$ docker run --rm -ti -e SSH_AUTH_SOCK=/ssh-agent -v $SSH_AUTH_SOCK:/ssh-agent -v /cache:/cache/gitsplit -v $PWD:/srv jderusse/gitsplit
```

# Sample with drone.io

Beware, the container have to push on your splited repository.
It could be a security issue. Use environments variables as defined in the official documentation

```yaml
# .gitsplit.yml
cache_url: "/cache/gitsplit"
splits:
  - prefix: "src/partA"
    target: "https://${GH_TOKEN}@github.com/my_company/project-partA.git"
origins:
  - ^master$
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
      - git fetch --prune --unshallow || true
      - gitsplit
```

# Sample with Travis CI

Beware, the container have to push on your splited repository.
It could be a security issue. Use environments variables as defined in the official documentation

```yaml
# .gitsplit.yml
cache_url: "/cache/gitsplit"
splits:
  - prefix: "src/partA"
    target: "https://${GH_TOKEN}@github.com/my_company/project-partA.git"
origins:
  - ^master$
```

```yaml
# .travis.yml
sudo: required
services:
  - docker
cache:
  directories:
    - /cache/gitsplit
install:
  - docker pull jderusse/gitsplit

  # update local repository. Because travis fetch a shallow copy
  - git config remote.origin.fetch "+refs/*:refs/*"
  - git config remote.origin.mirror true
  - git fetch --prune --unshallow || true

script:
  - docker run --rm -t -e GH_TOKEN -v /cache/gitsplit:/cache/gitsplit -v ${PWD}:/srv jderusse/gitsplit gitsplit --ref "${TRAVIS_BRANCH}"
```

# Sample with GitLab CI/CD

Beware, the container have to push on your splited repository.
It could be a security issue. Use environments variables as defined in the official documentation [GitLab SSH Deploy keys](https://docs.gitlab.com/ce/ssh/README.html#deploy-keys).

Note: I highly recommend to use ssh instead of https because of the username/password or username/token. Deploy keys are much easier to use with GitLab

```yaml
# .gitsplit.yml
cache_url: "cache/gitsplit"
splits:
  - prefix: "src/partA"
    target: "git@gitlab.com:my_company/project-partA.git"
origins:
  - ^master$
```

```yaml
# .gitlab-ci.yml with Docker runners
stages:
  - split

split:
  image: jderusse/gitsplit
  stage: split
  cache:
    key: "$CI_JOB_NAME-$CI_COMMIT_REF_NAME"
    paths:
      - cache/gitsplit
  variables:
    GIT_STRATEGY: clone
  before_script:
    - eval $(ssh-agent -s)
    - mkdir -p ~/.ssh
    - chmod 700 ~/.ssh
    - echo -e "Host *\n\tStrictHostKeyChecking no\n\n" > ~/.ssh/config
    - echo "$SSH_PRIVATE_KEY" | tr -d '\r' | ssh-add - > /dev/null
    - ssh-add -l
  script:
    - git config remote.origin.fetch "+refs/*:refs/*"
    - git config remote.origin.mirror true
    - git fetch --prune --unshallow || true
    - gitsplit --ref "${CI_COMMIT_REF_NAME}"
```
