package gitignore

const Template = `# Zyr Git Commit - generic .gitignore
# Keep intentional project files tracked; adjust this file for your stack.

# Operating systems
.DS_Store
.AppleDouble
.LSOverride
Thumbs.db
ehthumbs.db
Desktop.ini
$RECYCLE.BIN/

# Editors and IDEs
*.swp
*.swo
*~
.idea/
.vs/
.vscode/*
!.vscode/extensions.json
!.vscode/launch.json
!.vscode/settings.json
!.vscode/tasks.json
*.suo
*.user
*.userosscache
*.sln.docstates

# Local configuration and secrets
.env
.env.*
!.env.example
!.env.sample
!.env.template
*.pem
*.key
*.p12
*.pfx

# Logs, temporary files and caches
*.log
logs/
*.tmp
*.temp
.cache/
.sass-cache/

# Test and coverage output
coverage/
htmlcov/
.coverage
.coverage.*
.nyc_output/
TestResults/
*.lcov

# Common build output and compiled files
build/
dist/
out/
bin/
obj/
*.o
*.obj
*.so
*.dylib
*.dll
*.exe
*.class
*.py[cod]

# Python
__pycache__/
.pytest_cache/
.mypy_cache/
.ruff_cache/
.tox/
.nox/
.venv/
venv/
env/
pip-wheel-metadata/
*.egg-info/

# Node.js and frontend tools
node_modules/
.npm/
.yarn/cache/
.yarn/unplugged/
.pnpm-store/
npm-debug.log*
yarn-debug.log*
yarn-error.log*
pnpm-debug.log*
.next/
.nuxt/
.svelte-kit/

# Java and JVM
.gradle/
target/
*.jar
*.war
*.ear

# C and C++
CMakeFiles/
CMakeCache.txt
cmake-build-*/
Makefile.local

# .NET
artifacts/
packages/
*.nupkg

# Go
coverage.out
*.test

# Rust
target/
Cargo.lock.local

# Infrastructure and common tools
.terraform/
*.tfstate
*.tfstate.*
.serverless/
.aws-sam/
`
