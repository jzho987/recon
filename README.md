# recon

Recon is meant to be a simple dot file manager. Sometimes cloning a git repo and creating a sym-link is all you need.
Recon handles that for you with some simple addon features like syncing with remote, safely storing your old configs,
and more to come.

## Usage

Recon is all configured in the `recon.toml` configuratin file. You can declare a list of configs
to link up. After you properly configure your sources, simply run

```
recon sync
```

This will pull any new repositories and create the correct symlinks from the cloned
git repo to your `$HOME/.config/<config name>` directory.

**_NOTE:_**  recon advocates to be as dumb as possible, so it does not delete repositories cloned but no longer using. This is due to all the dotfiles being git repositories, if users added custom configurations and forgot to push then recon deletes their local copy which is undesirable. 

## Config

Example recon config:
```
[[repos]]
name = 'nvim'
remote = 'git@github.com:jzho987/nvim.git'
branch = 'new-packer' # use a selected branch

# use same dotfile repo for multiple configs
[[repos]]
name = 'alacritty'
remote = 'git@github.com:jzho987/dotfiles.git'
path = 'alacritty'

[[repos]]
name = 'starship'
remote = 'git@github.com:jzho987/dotfiles.git'
path = 'starship'

[[repos]]
name = 'aerospace'
remote = 'git@github.com:jzho987/dotfiles.git'
path = 'aerospace'
```
