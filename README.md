> **CLI tool · Zero config · No dependencies · No database · No cloud · If kubectl works, steered works.**

https://github.com/user-attachments/assets/51420bb5-cf55-43fa-ab88-991d38388304

## what is steered?

You just joined a team, landed a contract, inherited an unknown kubernetes cluster, or need to troubleshoot fast. One command. Zero setup. steered watches your cluster live, thinks like a senior Kubernetes engineer, tells you exactly what to fix, and confirms when you fixed it.


# steered

> the light that guides you through darkness.

**[steered.dev](https://steered.dev)  · [demo](https://steered.dev/#demo)**

`steered` is a live Kubernetes cluster intelligence tool for engineers who need the complete picture fast. No agents, no SaaS, no cloud dependency. Just run `steered` and everything becomes clear.


## install

**linux / mac**
```bash
curl -fsSL https://steered.dev/install | sudo sh
```


## how it works

```
$ steered

  live view loads. your cluster, fully mapped.
  health score updates every 30 seconds.

  must fix      — things breaking right now, with exact commands
  good practice — what will hurt you later
  security      — what you should not have left open

  hit 'a' when you want to understand why

    WHY    the root cause, no jargon
    ACTION what to do about it
    RUN    the exact command, ready to copy
    RISK   what happens if you ignore it

  hit 'esc' to go back to the live view

  when you fix something, steered notices.  ✓
```
---

## features

- live terminal UI — probes cluster every 30 seconds
- complete cluster snapshot — nodes, namespaces, pods, deployments, services, ingresses, pvcs
- health percentage — real time cluster health score
- must fix — critical issues with exact kubectl commands
- good practice — cost and stability recommendations
- security audit — privileged containers, missing network policies, exposed secrets, unpinned images
- cost signals — idle namespaces, unattached pvcs, missing resource limits
- ai powered analysis — press 'a' for guided fix with WHY, ACTION, RUN, RISK
- supports openai, anthropic, and local ollama
- air-gapped friendly — local ollama means zero data leaves your machine
- zero cluster footprint — no agent, no install, no SaaS
- works on any cluster — eks, gke, aks, kubeadm, k3s, minikube, orbstack

---



**go install**
```bash
go install github.com/steereddev/steered@latest
```

---

## usage

```bash
steered                        # start live cluster monitoring
steered --context staging      # target different cluster
steered --kubeconfig ~/my.cfg  # explicit kubeconfig
steered --setup                # configure LLM provider
steered --clear                # clear saved config and API keys
```

**inside steered**
```
'a'      → open AI analysis view
'esc'    → return to main view
ctrl+c   → exit
```

---

## llm setup

on first run steered will ask you to configure a language model:

```
[1] ollama      local, free, private, recommended
[2] openai      best quality, requires API key
[3] anthropic   best reasoning, requires API key
[4] skip        snapshot only, configure later
```

API keys are stored locally with a 24 hour expiry and never leave your machine.

reconfigure anytime:
```bash
steered --setup
```

---

## how it connects

steered follows the standard kubeconfig precedence — exactly like kubectl:

```
1. --kubeconfig flag    explicit override
2. KUBECONFIG env var   ci/cd systems
3. ~/.kube/config       default fallback
```

no configuration needed. if kubectl works, steered works.

steered talks directly to the Kubernetes API server using client-go — the same library kubectl uses. it is not a kubectl wrapper.

---

## why steered?

| | steered | kubecost | cast.ai | k9s |
|---|---|---|---|---|
| install | single binary | helm chart | saas agent | binary |
| live monitoring | yes | no | yes | yes |
| ai guided fix | yes | no | no | no |
| data privacy | local only | cloud | cloud | local |
| air-gapped | yes | no | no | yes |
| security audit | yes | no | no | no |
| cost signals | yes | yes | yes | no |
| open source | yes | partial | no | yes |
| zero footprint | yes | no | no | yes |

---

## author

built by [Awet Tsegazeab](https://github.com/steereddev) — Senior Golang Engineer, Linux & CKA Certified

> your existence alone reveals all.

---

https://github.com/user-attachments/assets/2c3d8d5f-17d4-4d6a-8dbc-318b679b2220

## license

MIT