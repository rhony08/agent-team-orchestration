#!/bin/bash
# setup-todo-project.sh
# Creates a sample todo-list project for testing orchestration

set -e

PROJECT_DIR="${1:-$HOME/todo-project}"

echo "Creating todo-list project at $PROJECT_DIR..."

# Create directories
mkdir -p "$PROJECT_DIR"/{todo-api,todo-web,todo-db}

# --- Setup todo-api ---
echo "Setting up todo-api..."
cd "$PROJECT_DIR/todo-api"
git init -q

cat > package.json << 'EOF'
{
  "name": "todo-api",
  "version": "1.0.0",
  "scripts": {
    "dev": "ts-node src/index.ts",
    "build": "tsc"
  },
  "dependencies": {
    "express": "^4.18.2",
    "@prisma/client": "^5.0.0",
    "cors": "^2.8.5"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "ts-node": "^10.9.1",
    "@types/express": "^4.17.17",
    "@types/cors": "^2.8.13",
    "@types/node": "^20.0.0",
    "prisma": "^5.0.0"
  }
}
EOF

cat > tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "commonjs",
    "lib": ["ES2020"],
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true
  },
  "include": ["src/**/*"]
}
EOF

mkdir -p src/routes src/models src/middleware
cat > src/index.ts << 'EOF'
import express from 'express';
import cors from 'cors';
import todoRoutes from './routes/todos';

const app = express();
const PORT = process.env.PORT || 4000;

app.use(cors());
app.use(express.json());
app.use('/api/todos', todoRoutes);

app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

app.listen(PORT, () => {
  console.log(`Todo API running on port ${PORT}`);
});
EOF

cat > src/routes/todos.ts << 'EOF'
import { Router, Request, Response } from 'express';

const router = Router();

// TODO: Implement with Prisma client
// For now, return placeholder responses

router.get('/', async (req: Request, res: Response) => {
  res.json({ message: 'List todos - implement with Prisma' });
});

router.get('/:id', async (req: Request, res: Response) => {
  res.json({ message: `Get todo ${req.params.id} - implement with Prisma` });
});

router.post('/', async (req: Request, res: Response) => {
  res.json({ message: 'Create todo - implement with Prisma', body: req.body });
});

router.patch('/:id', async (req: Request, res: Response) => {
  res.json({ message: `Update todo ${req.params.id} - implement with Prisma` });
});

router.delete('/:id', async (req: Request, res: Response) => {
  res.json({ message: `Delete todo ${req.params.id} - implement with Prisma` });
});

export default router;
EOF

cat > .gitignore << 'EOF'
node_modules/
dist/
.env
EOF

git add -A && git commit -q -m "Initial commit: Express API skeleton"


# --- Setup todo-web ---
echo "Setting up todo-web..."
cd "$PROJECT_DIR/todo-web"
git init -q

cat > package.json << 'EOF'
{
  "name": "todo-web",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build"
  },
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0"
  },
  "devDependencies": {
    "@types/react": "^18.2.0",
    "@types/react-dom": "^18.2.0",
    "@vitejs/plugin-react": "^4.0.0",
    "typescript": "^5.0.0",
    "vite": "^5.0.0"
  }
}
EOF

cat > tsconfig.json << 'EOF'
{
  "compilerOptions": {
    "target": "ES2020",
    "useDefineForClassFields": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true
  },
  "include": ["src"]
}
EOF

mkdir -p src/components src/hooks src/api src/types
cat > src/App.tsx << 'EOF'
import { useState } from 'react';

function App() {
  const [todos, setTodos] = useState([]);

  return (
    <div>
      <h1>Todo List</h1>
      {/* TODO: Implement TodoList, AddTodo components */}
      <p>Implement the todo app here</p>
    </div>
  );
}

export default App;
EOF

cat > src/main.tsx << 'EOF'
import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
EOF

cat > index.html << 'EOF'
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Todo App</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
EOF

cat > vite.config.ts << 'EOF'
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:4000',
    },
  },
});
EOF

cat > .gitignore << 'EOF'
node_modules/
dist/
.env
EOF

git add -A && git commit -q -m "Initial commit: React + Vite skeleton"


# --- Setup todo-db ---
echo "Setting up todo-db..."
cd "$PROJECT_DIR/todo-db"
git init -q

cat > package.json << 'EOF'
{
  "name": "todo-db",
  "version": "1.0.0",
  "scripts": {
    "migrate": "prisma migrate dev",
    "generate": "prisma generate",
    "studio": "prisma studio"
  },
  "dependencies": {
    "@prisma/client": "^5.0.0"
  },
  "devDependencies": {
    "prisma": "^5.0.0"
  }
}
EOF

mkdir -p prisma
cat > prisma/schema.prisma << 'EOF'
// Prisma Schema for Todo App
// TODO: Implement the Todo model

generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

// Define your Todo model here
// model Todo {
//   id          String   @id @default(uuid())
//   title       String
//   description String?
//   completed   Boolean  @default(false)
//   createdAt   DateTime @default(now())
//   updatedAt   DateTime @updatedAt
// }
EOF

cat > .env.example << 'EOF'
DATABASE_URL="postgresql://user:password@localhost:5432/todo_db"
EOF

cat > .gitignore << 'EOF'
node_modules/
.env
EOF

git add -A && git commit -q -m "Initial commit: Prisma schema skeleton"


echo ""
echo "✓ Todo project created at $PROJECT_DIR"
echo ""
echo "Repositories:"
echo "  • $PROJECT_DIR/todo-api  (Express API)"
echo "  • $PROJECT_DIR/todo-web  (React Frontend)"
echo "  • $PROJECT_DIR/todo-db   (Prisma Database)"
echo ""
echo "Next steps:"
echo "  cd $PROJECT_DIR"
echo "  crush-orchestrator init todo-app --repos ./todo-api,./todo-web,./todo-db"
echo "  crush-orchestrator start"
