# A Docker image with git and splitsh-lite

See [the official site](https://github.com/splitsh/lite) for more information about splitsh.

# Usage

With drone.io

```yaml
workspace:
  base: /drone

pipeline:
  split:
    image: jderusse/gitsplit
    pull: true
    commands:
      - git clone file:///drone/mono /drone/mono
      - cd /drone/mono
      - git remote add splitA git@github.com:my-company/project-splitA.git

      - git branch -f splitA-$DRONE_REPO_BRANCH `splitsh-lite --prefix=splitA/`
      - git push splitA splitA-$DRONE_REPO_BRANCH:$DRONE_REPO_BRANCH

```
