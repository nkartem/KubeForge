# Быстрый старт с KubeForge

Эта инструкция поможет вам запустить KubeForge и создать ваш первый Kubernetes кластер за несколько минут.

## Предварительные требования

### На машине с KubeForge:
- Go 1.21+ (для сборки из исходников)
- Git (для клонирования репозитория)

### На целевых хостах (будущие ноды Kubernetes):
- Ubuntu 20.04/22.04 или Debian 11/12
- Минимум 2GB RAM, 2 CPU, 20GB диска
- SSH доступ с ключом (без пароля)
- Sudo без пароля для пользователя
- Открытые порты:
  - 6443 (Kubernetes API)
  - 2379-2380 (etcd)
  - 10250-10252 (kubelet, scheduler, controller)
  - 30000-32767 (NodePort сервисы)

## Шаг 1: Клонирование и сборка

```bash
# Клонируем репозиторий
git clone https://github.com/yourusername/kubeforge.git
cd kubeforge

# Скачиваем зависимости
go mod download

# Собираем проект
make build

# Или просто запускаем без сборки
go run ./cmd/kubeforge-server
```

## Шаг 2: Настройка SSH ключей

Убедитесь, что у вас есть SSH ключ для доступа к целевым хостам:

```bash
# Создать новый SSH ключ (если нужно)
ssh-keygen -t rsa -b 4096 -f ~/.ssh/kubeforge_rsa

# Скопировать ключ на все целевые хосты
ssh-copy-id -i ~/.ssh/kubeforge_rsa.pub ubuntu@192.168.1.10
ssh-copy-id -i ~/.ssh/kubeforge_rsa.pub ubuntu@192.168.1.11
# ... и так далее для всех нод

# Проверить доступ
ssh -i ~/.ssh/kubeforge_rsa ubuntu@192.168.1.10 'sudo echo "OK"'
```

## Шаг 3: Подготовка конфигурации кластера

Отредактируйте файл `examples/cluster-example.json`:

```json
{
  "name": "my-first-cluster",
  "k8s_version": "1.28.0",
  "cni": "calico",
  "container_runtime": "containerd",
  "control_planes": [
    {
      "hostname": "cp1",
      "address": "192.168.1.10",
      "user": "ubuntu",
      "ssh_key_path": "/home/user/.ssh/kubeforge_rsa",
      "port": 22
    }
  ],
  "workers": [
    {
      "hostname": "worker1",
      "address": "192.168.1.11",
      "user": "ubuntu",
      "ssh_key_path": "/home/user/.ssh/kubeforge_rsa",
      "port": 22
    }
  ]
}
```

## Шаг 4: Запуск KubeForge

```bash
# В одном терминале запустите сервер
./bin/kubeforge

# Или через make
make run
```

Сервер запустится на `http://localhost:8080`

## Шаг 5: Создание кластера

Отправьте запрос на создание кластера:

```bash
cd examples

# Используя curl
curl -X POST http://localhost:8080/api/clusters \
  -H "Content-Type: application/json" \
  -d @cluster-example.json

# Или используя готовый скрипт
chmod +x create-cluster.sh
./create-cluster.sh
```

## Шаг 6: Мониторинг прогресса

Следите за статусом создания кластера:

```bash
# Получить список кластеров
curl http://localhost:8080/api/clusters | jq

# Получить детали конкретного кластера (ID=1)
curl http://localhost:8080/api/clusters/1 | jq

# Получить события/логи provision
curl http://localhost:8080/api/clusters/1/events | jq
```

Процесс создания кластера занимает **10-20 минут** в зависимости от:
- Скорости интернета (скачивание пакетов)
- Количества нод
- Производительности хостов

## Шаг 7: Получение kubeconfig

Когда статус кластера станет `ready`, скачайте kubeconfig:

```bash
# Скачать kubeconfig
curl http://localhost:8080/api/clusters/1/kubeconfig -o kubeconfig.yaml

# Использовать его
export KUBECONFIG=$(pwd)/kubeconfig.yaml

# Проверить ноды
kubectl get nodes

# Проверить pods
kubectl get pods -A
```

## Шаг 8: Установка приложений

Теперь ваш кластер готов! Устанавливайте приложения:

```bash
# Пример: установка NGINX
kubectl create deployment nginx --image=nginx
kubectl expose deployment nginx --port=80 --type=NodePort

# Проверить сервис
kubectl get svc nginx
```

## Troubleshooting

### Проблема: SSH connection failed

**Решение:**
```bash
# Проверьте доступность хоста
ping 192.168.1.10

# Проверьте SSH
ssh -i ~/.ssh/kubeforge_rsa ubuntu@192.168.1.10

# Проверьте права на ключ
chmod 600 ~/.ssh/kubeforge_rsa
```

### Проблема: Swap is enabled

**Решение:**
```bash
# На каждом хосте выполните:
sudo swapoff -a
sudo sed -i '/ swap / s/^/#/' /etc/fstab
```

### Проблема: kubeadm init failed

**Решение:**
```bash
# Проверьте логи в API
curl http://localhost:8080/api/clusters/1/events | jq '.data[] | select(.level=="error")'

# На хосте проверьте логи kubelet
sudo journalctl -u kubelet -f

# Сбросить и попробовать снова
sudo kubeadm reset -f
```

### Проблема: Pods в состоянии Pending

**Решение:**
CNI плагин может не установиться автоматически. Установите вручную:

```bash
# Для Calico
kubectl apply -f https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml

# Для Flannel
kubectl apply -f https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml
```

## Полезные команды

```bash
# Просмотр всех кластеров
curl http://localhost:8080/api/clusters | jq

# Удаление кластера
curl -X DELETE http://localhost:8080/api/clusters/1

# Проверка здоровья API
curl http://localhost:8080/healthz

# Сборка Docker образа
make docker-build

# Запуск в Docker
docker run -p 8080:8080 -v $(pwd)/kubeforge.db:/app/data/kubeforge.db kubeforge:latest
```

## Следующие шаги

1. Настройте мониторинг (Prometheus + Grafana)
2. Установите Ingress Controller (NGINX/Traefik)
3. Настройте автоматические backup
4. Добавьте дополнительные worker ноды
5. Настройте CI/CD pipeline

## Дополнительная информация

- [Полная документация](README.md)
- [Примеры конфигураций](examples/)
- [API Reference](docs/api.md) (TODO)
- [Архитектура](docs/architecture.md) (TODO)

Удачи с KubeForge! 🚀
