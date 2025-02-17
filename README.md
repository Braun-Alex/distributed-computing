# Distributed Miller-Rabin algorithm

## Introduction

The project implements a **distributed version of the Miller-Rabin primality test** using the **heterogeneous PARCS** technology. The system consists of a **master node** (implemented in Go) and multiple **worker nodes** (implemented in Python), enabling parallel execution of the algorithm.

## Technologies

- **PARCS**: a framework for distributed computing.
- **Python**: used for worker node implementation.
- **Go**: used for master node implementation.
- **Docker & Docker Swarm**: for containerized deployment.

## Miller-Rabin algorithm

The Miller-Rabin test is a probabilistic algorithm to determine whether a number `n` is prime.

1. Express `n - 1` in the form `2^s * d`, where:
   - `s` is a non-negative integer,
   - `d` is an odd integer.

2. Choose a random integer `a` such that `1 < a < n - 1`.

3. If one of the following conditions holds:
   - `a^d ≡ 1 (mod n)`
   - `a^(2^r * d) ≡ -1 (mod n)`
   for some `0 ≤ r < s`, then `n` is *probably prime*.

4. If neither of the above conditions is met, then `n` is *composite*.

5. Repeat the test `k` times with different values of `a` to increase the probability that `n` is truly prime.

The probability of falsely identifying a composite number as prime is at most `4^(-k)`.

## Distributed implementation

The algorithm is **distributed** across multiple worker nodes to enhance performance.

### **Worker nodes**
- Receive test parameters from the master node.
- Perform iterations of the Miller-Rabin test.
- Return the result to the master node.
- If a composite witness is found, the worker halts all tests.

### **Master node**
- Receives input values `n` and iteration count `k`.
- Computes values of `s` and `d`.
- Distributes the work among worker nodes.
- Collects results and determines primality.

## Cluster management

One of the biggest advantages of this PARCS implementation is that it does not require any special setup. Any Docker Swarm cluster will do.

### Swarm cluster on Google Cloud Platform

1. First of all you need to make sure that `gcloud` is linked to your accont. If it's the first time you use the CLI just fire `gcloud init`
and follow the instructions. I am also gonna set up the sensible defaults for my region via `gcloud config set compute/zone us-west1-b`.

2. Now let's start a couple of VMs that are will form a cluster later. Here I am creating a cluster of 4 nodes that will be managed by a leader.

    ```console
    me@laptop~:$ gcloud compute instances create leader worker-1 worker-2 worker-3
    Created [https://www.googleapis.com/compute/v1/projects/ember-27/zones/us-west1-b/instances/leader].
    Created [https://www.googleapis.com/compute/v1/projects/ember-27/zones/us-west1-b/instances/worker-1].
    Created [https://www.googleapis.com/compute/v1/projects/ember-27/zones/us-west1-b/instances/worker-2].
    Created [https://www.googleapis.com/compute/v1/projects/ember-27/zones/us-west1-b/instances/worker-3].
    
    NAME      ZONE        MACHINE_TYPE   PREEMPTIBLE  INTERNAL_IP  EXTERNAL_IP     STATUS
    leader    us-west1-b  n1-standard-1               10.138.0.6   35.247.55.235   RUNNING
    worker-1  us-west1-b  n1-standard-1               10.138.0.8   35.233.219.127  RUNNING
    worker-2  us-west1-b  n1-standard-1               10.138.0.7   34.83.142.137   RUNNING
    worker-3  us-west1-b  n1-standard-1               10.138.0.5   35.247.116.107  RUNNING
    ```

3. Unfortunatelly the default Debian image does not ship Docker by default, but we can use this [convenience script][convenience-script] to install
the engine as follows

    ```console
    me@laptop~:$ gcloud compute ssh leader
    me@leader~:$ curl -fsSL https://get.docker.com | sudo sh
    ```

    Make sure that you do this step for every node in the cluster replacing `leader` with a corresponding name.

4. It is time to initialize a swarm. We can do this by `ssh`-ing into a `leader` and running commands:

    ```console
    me@laptop~:$ gcloud compute ssh leader
    me@leader~:$ sudo docker swarm init
    Swarm initialized: current node (p7ywd9wbh6th1hy6t5hlsqv0w) is now a manager.
    
    To add a worker to this swarm, run the following command:
    
        docker swarm join --token \
          SWMTKN-1-4cj55yg229l3updnigyz86p63x9bb599htytlmtbhulo4m633d-4kcfduodzvitw4y52flh19g32 \
          10.138.0.6:2377
    ```

5. Having a `join-token` from the previous step we can connect `worker` nodes to a `leader` like follows:

    ```console
    me@laptop~:$ gcloud compute ssh worker-1
    me@worker-1~:$ sudo docker swarm join --token \
             SWMTKN-1-4cj55yg229l3updnigyz86p63x9bb599htytlmtbhulo4m633d-4kcfduodzvitw4y52flh19g32 \
             10.138.0.6:2377
    
    This node joined a swarm as a worker.
    ```

    Do not forget to do this step **for each one** of the `worker` nodes you created.

6. **IMPORTANT!** PARCS needs `leader`-s Docker Engine to listen on the port `4321`.

    This is the only extra step that users have to take to be able to run PARCS on a
    barebones Docker Swarm cluster. Here are commands that do exactly that

    ```console
    me@laptop~:$ gcloud compute ssh leader
    me@leader~:$ sudo sed -i '/ExecStart/ s/$/ -H tcp:\/\/0.0.0.0:4321/' \
                        /lib/systemd/system/docker.service
    me@leader~:$ sudo systemctl daemon-reload
    me@leader~:$ sudo systemctl restart docker
    ```
    
7. **IMPORTANT!** PARCS is also utilizing a custom overlay network that one can create by typing:

    ```console
    me@laptop~:$ gcloud compute ssh leader
    me@leader~:$ sudo docker network create -d overlay parcs
    ```

Now we have a fully configured Docker Swarm cluster ready to run PARCS services.

### PARCS modules

All the PARCS modules (aka services) are accessible from the Docker registry. All the code can be found in this repo
under `parcs-nw-py/` and `parcs-nw-go/` subdirs.

#### Service

Now assuming the code lives in the file `worker.py` we can build a Docker image for this program by running:

```console
me@laptop~:$ wget https://raw.githubusercontent.com/Braun-Alex/distributed-computing/main/parcs-nw-py/Dockerfile
me@laptop~:$ cat Dockerfile
FROM lionell/parcs-py

COPY worker.py .
CMD [ "python3", "worker.py" ]
me@laptop~:$ docker build -t oleksiibraun/parcs-nw-py .
me@laptop~:$ docker push oleksiibraun/parcs-nw-py:latest
```

PARCS provides base Docker images for all supported languages: [lionell/parcs-py][parcs-py], [lionell/parcs-go][parcs-go].

#### Runner

PARCS needs a special type of jobs that will kick off the computation. These are **Runners** and they can be
implemented in a very similar way to a services.

Assuming the code lives in the file `master.go` we can build a Docker image for this program by running:

```console
me@laptop~:$ wget https://raw.githubusercontent.com/Braun-Alex/distributed-computing/main/parcs-nw-go/Dockerfile
me@laptop~:$ cat Dockerfile
FROM lionell/parcs-go

COPY master.go .
CMD [ "go", "run ", "master.go" ]
me@laptop~:$ docker build -t oleksiibraun/parcs-nw-go .
me@laptop~:$ docker push oleksiibraun/parcs-nw-go:latest
```

### Running PARCS modules

In order to run a PARCS runner on a cluster you need to know **internal IP of the leader**. It can be obtained from
the Google Compute Engine UI or by firing this command:

```console
me@laptop~:$ gcloud compute instances list | grep leader | awk '{print "tcp://" $4 ":4321"}'
tcp://10.138.0.6:4321
```

Now to start a service just do this:

```console
me@laptop~:$ gcloud compute ssh leader
me@leader~:$ sudo docker service create \
                    --network parcs \
                    --restart-condition none \
                    --env LEADER_URL=tcp://<LEADER INTERNAL IP>:4321 \
                    --name runner \
                    --env A=1000000000000000003 \
                    --env WORKERS=3 \
                    --env ITERATIONS=50 \
                    oleksiibraun/parcs-nw-go
me@leader~:$ sudo docker service logs -f runner
me@leader~:$ sudo docker service rm runner
```

### Last step

Do not forget to destroy all created VMs. If you do not do it Google Cloud Platform can charge you!

```console
me@laptop~:$ gcloud compute instances delete leader worker-1 worker-2 worker-3
```

## License

This project is licensed under the MIT License. See the LICENSE file for details.
