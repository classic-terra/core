Generally we should avoid shell scripting and write tests purely in Golang.
However, some libraries are not Goroutine-safe (e.g. app simulations cannot be run safely in parallel),
and OS-native threading may be more efficient for many parallel simulations, so we use shell scripts here.

# For quick node testing

```
bash scripts/run-node.sh <terrad binary> <denom>
```

If "terrad binary" and "denom" is left blank, it will build current terrad and set denom to uluna.
```
bash scripts/run-node.sh
```

# For quick upgrade testing

Open scripts/upgrade-test.sh and change:
* OLD_VERSION env to the tag that you want (currently it is 1.0.4)
* SOFTWARE_UPGRADE_NAME env to the upgrade version that you want (currently it is v2)

Require:
* screen command that has "-L" and "-Logfile" flag: "screen -h"

This will start upgrade test from terrad OLD_VERSION to current branch 
```
bash scripts/upgrade-test.sh
```

This will re install old binary for debugging terrad OLD_VERSION
```
bash scripts/upgrade-test.sh --reinstall-old
```