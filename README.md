# KubeForge

**KubeForge** - это мощный инструмент для автоматизации создания и управления Kubernetes кластерами на выбранных хостах с веб-интерфейсом.

## Возможности

- ✅ Создание Kubernetes кластеров через **kubeadm** на bare-metal или VM
- ✅ Поддержка нескольких control plane нод (HA setup)
- ✅ Автоматическая установка container runtime (containerd, cri-o)
- ✅ Поддержка различных CNI плагинов (Calico, Flannel, Weave, Cilium)
- ✅ Добавление и удаление worker нод
- ✅ Веб API для управления кластерами
- ✅ Хранение kubeconfig и метаданных кластеров
- ✅ Логирование всех операций provision
- 🚧 Веб UI (в разработке)
- 🚧 Управление кластерами через kubectl (port-forward, exec, logs)

## Архитектура

```
kubeforge/
├── cmd/
│   └── kubeforge-server/      # Backend server
├── internal/
│   ├── api/                   # REST API handlers
│   ├── db/                    # Database models (GORM)
│   ├── provision/             # Provisioning logic
│   │   ├── iface.go           # Provisioner interface
│   │   ├── kubeadm.go         # Kubeadm provisioner
│   │   ├── ssh_client.go      # SSH utilities
│   │   └── types.go           # Data types
│   └── config/                # Configuration
└── go.mod
```

## Быстрый старт

### 1. Установка зависимостей

```bash
go mod download
```

### 2. Сборка

```bash
make build
```

### 3. Запуск сервера

```bash
./bin/kubeforge
```

Сервер запустится на `http://localhost:8080`

### 4. Создание кластера

```bash
curl -X POST http://localhost:8080/api/clusters \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-cluster",
    "k8s_version": "1.28.0",
    "cni": "calico",
    "container_runtime": "containerd",
    "control_planes": [
      {
        "hostname": "cp1",
        "address": "192.168.1.10",
        "user": "ubuntu",
        "ssh_key_path": "/path/to/ssh/key",
        "port": 22
      }
    ],
    "workers": [
      {
        "hostname": "worker1",
        "address": "192.168.1.11",
        "user": "ubuntu",
        "ssh_key_path": "/path/to/ssh/key",
        "port": 22
      }
    ]
  }'
```

### 5. Получение списка кластеров

```bash
curl http://localhost:8080/api/clusters
```

### 6. Скачивание kubeconfig

```bash
curl http://localhost:8080/api/clusters/1/kubeconfig -o kubeconfig.yaml
export KUBECONFIG=kubeconfig.yaml
kubectl get nodes
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | Health check |
| GET | `/api/clusters` | List all clusters |
| POST | `/api/clusters` | Create new cluster |
| GET | `/api/clusters/:id` | Get cluster details |
| DELETE | `/api/clusters/:id` | Delete cluster |
| GET | `/api/clusters/:id/kubeconfig` | Download kubeconfig |
| GET | `/api/clusters/:id/events` | Get cluster events |
| POST | `/api/clusters/:id/nodes` | Add node to cluster |
| DELETE | `/api/clusters/:id/nodes/:nodeId` | Remove node |

## Переменные окружения

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
DB_DRIVER=sqlite           # sqlite, postgres, mysql
DB_DSN=kubeforge.db

# Logging
LOG_LEVEL=info
LOG_FORMAT=console
```

## Требования к хостам

Для успешного создания кластера хосты должны удовлетворять следующим требованиям:

- **OS**: Ubuntu 20.04/22.04, Debian 11/12, CentOS 8+, RHEL 8+
- **RAM**: Минимум 2GB (рекомендуется 4GB+ для control plane)
- **CPU**: Минимум 2 cores
- **Disk**: Минимум 20GB свободного места
- **Network**: Все ноды должны иметь связь друг с другом
- **SSH**: Доступ по SSH с ключом (без пароля)
- **Sudo**: Пользователь должен иметь sudo без пароля

## Установка зависимостей на хостах

KubeForge автоматически установит все необходимое, но вы можете подготовить хосты вручную:

```bash
# Отключить swap
sudo swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab

# Установить необходимые пакеты
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl
```

## Разработка

### Сборка

```bash
make build
```

### Запуск в dev режиме

```bash
make run
```

### Тесты

```bash
make test
```

### Docker образ

```bash
make docker-build
```

## Roadmap

- [x] Базовая архитектура
- [x] Kubeadm provisioner
- [x] API для создания кластеров
- [x] Поддержка containerd
- [ ] Веб UI (React/Vue)
- [ ] WebSocket для realtime логов
- [ ] Поддержка k3s provisioner
- [ ] Поддержка Ansible provisioner
- [ ] Управление через kubectl (exec, port-forward)
- [ ] Backup и restore кластеров
- [ ] Мониторинг кластеров (Prometheus integration)
- [ ] RBAC и multi-tenancy
- [ ] Cloud provider интеграция (AWS, Azure, GCP)

## Технологии

- **Backend**: Go 1.21+
- **Web Framework**: gorilla/mux
- **Database**: GORM (SQLite, PostgreSQL, MySQL)
- **SSH**: golang.org/x/crypto/ssh
- **Kubernetes**: kubeadm
- **Container Runtime**: containerd, cri-o

## Лицензия

MIT

## Вклад в проект

Pull requests приветствуются! Для крупных изменений пожалуйста создайте issue для обсуждения.

## Поддержка

Если у вас возникли вопросы или проблемы, создайте issue в GitHub репозитории.
