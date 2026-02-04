# Storage & Data Layout - Questions

- Q: What are the allowed characters/length limits for task slugs, and what is the collision strategy?
  A: Slugify to lowercase [a-z0-9-], max 48 chars; on collision append -<4char> hash.
