# FocusBoard TUI

I spend a lot of time in the terminal. After trying different tools for keeping track of my projects, I always ended up dumping everything into random markdown files scattered across my computer.

Since I started learning Go, I figured building a TUI app for project management would be a perfect way to practice. So here we are.

## Features

- Multiple projects with separate todo lists
- Add, edit, and delete projects/todos
- Mark todos as complete/incomplete
- Attach links to todos (I use this mainly to link tasks with PRs when I want to check later why I made certain decisions)
- Everything stored in a local JSON file

## Installation

```bash
go install github.com/danjecu/focusboard-tui@latest
```

Or build from source:

```bash
git clone https://github.com/danjecu/focusboard-tui.git
cd focusboard-tui
go build -o focusboard-tui
```

## Usage

```bash
./focusboard-tui
```

Data gets saved to `todos.json` in whatever directory you run it from.

## Key Bindings

| Key | Action |
|-----|--------|
| `j/k` or `↓/↑` | Navigate |
| `ctrl+h/l` or `←/→` | Switch between projects and todos |
| `Enter` | Open project / Toggle todo |
| `a` | Add |
| `e` | Edit |
| `d` | Delete |
| `l` | Set link |
| `o` | Open link |
| `q` | Quit |

## What's Next

- [ ] GitHub integration - sync todos with issues/PRs
- [ ] Due dates for todos
- [ ] Priority levels (high/medium/low)
- [ ] Search/filter functionality
- [ ] Tags/categories for better organization
- [ ] Export to markdown

## License

MIT
