# Runner & Orchestration - Questions

- Q: What locking/coordination mechanism should prevent two run-task instances for the same task, and how are stale locks recovered?
  A: We create unique folders with date and PID. Each task must make sure it creates the folder, and such folder was not existing before. Otherwise, try again and hope timestamp and PID will make it unique and different.
