# Monny

Monny is a proactive monitoring tool for your application that detects problems without having to write complicated alerting rules

## Goals

**No configuration** - No YAML config files, no need to pre-register your metrics, no alerting rules to define.  It scans your structured log output to find your metrics, monitors them, and sends an alert when something changes (and also figures out when something returns back to normal on its own)

**Stats are better than graphs** - We know how things like latency and error rates can be modeled statistically.  Monny models your metrics statistically to detect when a change in these metrics is significant, without the need to look at graphs or figure out the alerting rule yourself.  It can detect small changes that would lead to lots of false alarms if done in something like Prometheus.

**Simple deployment** - Single binary client and server that reads your application logs and finds your metrics automatically.  It can monitor things like latency, distributed traces, memory consumption, CPU utilization, and error rates.  Works with Kubernetes, bare metal, Docker, and whatever comes next.  No external database required, making it easy to run yourself.

**Advanced alerting** - Send alerts to email, text, Slack, and many more.  Get alerts when something needs human intervention.  Only want an alert when less than 2 of 5 processes are functioning normally?  No problem.  Want to silence an alert, snooze it, or send it to someone else?  It has an email or slack based workflow to deal with alerts right where you get them.

**Only the context you need** - Alerts aren't just metrics, but come with log context so you can see what led up to the alert.  You don't need to run ELK plus Prometheus, it's all combined together in an intuitive UI so you can figure out what's wrong, fix it, and get back to what you were doing.

## Beta testers needed

Want to help make sure Monny addresses your application monitoring wishlist?  We need beta testers to run it on non-production workloads to provide feedback and collect data on the performance of the statistical models.  Beta testing starts in March 2020.

If you have a publically available email address on Github, just star this repository and we'll be in touch.  Otherwise, just drop your email here: [Beta Opt-in](https://forms.gle/HoCaqG7qC24aaLpKA)
 
