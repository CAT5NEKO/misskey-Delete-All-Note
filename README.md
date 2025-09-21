## Misskey Note Delete Tool

A simple CLI tool to bulk-delete your posts on Misskey.  
It designed for server environments and can be used with scheduled tasks like cron. 🚀

## Caution

This environment is intended for local-only posts.  
Since Fediverse platforms propagate delete requests across servers, excessive deletions may cause unnecessary load.  
When using this tool on federated posts, set long intervals between requests and design your 
code to minimise server load, even where no rate limits apply.

---

## 🚀 Getting Started

1. Create a `.env` file based on `.env.example`, and fill in your Misskey API token and host.
2. Build the tool.

```bash
   go build
```
3. Run it

```bash
./misskeyNotedel
```

🧫 Make sure your .env includes a valid API token and Misskey host (e.g., misskey.hoge).

🐈 For Developers

This tool was originally built as a way for me to get more hands-on experience with Go.  
If you spot anything that could be improved, feel free to open an issue or send a PR.  
Suggestions and contributions are always welcome! 💥

