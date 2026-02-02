# Linux Deep Dive Guide - From Boot to Security

A comprehensive guide to understanding Linux internals, from the moment you press the power button to advanced security concepts.

---

## Table of Contents

1. [Boot Process & Bootloader](#1-boot-process--bootloader)
2. [Linux Kernel](#2-linux-kernel)
3. [Processes & Threads](#3-processes--threads)
4. [Systemd & Init System](#4-systemd--init-system)
5. [File System](#5-file-system)
6. [User Management](#6-user-management)
7. [Security](#7-security)

---

## 1. Boot Process & Bootloader

### 1.1 Overview of the Boot Sequence

When you press the power button, the following sequence occurs:

```
Power On → BIOS/UEFI → Bootloader → Kernel → Init System → User Space
```

### 1.2 BIOS vs UEFI

#### BIOS (Basic Input/Output System)
- Legacy firmware interface (since 1981)
- Stored in ROM chip on motherboard
- 16-bit real mode
- Maximum addressable memory: 1MB
- Uses MBR (Master Boot Record) partitioning
- Maximum disk size: 2TB
- Boot process is sequential

```
BIOS Boot Sequence:
1. POST (Power-On Self-Test)
2. Load first sector (512 bytes) from boot device
3. Execute bootloader from MBR
4. MBR loads second-stage bootloader
5. Bootloader loads kernel
```

#### UEFI (Unified Extensible Firmware Interface)
- Modern replacement for BIOS
- 32-bit or 64-bit mode
- Can address more memory
- Uses GPT (GUID Partition Table)
- Supports disks > 2TB
- Faster boot times
- Secure Boot capability

```
UEFI Boot Sequence:
1. SEC (Security Phase) - Initialize CPU
2. PEI (Pre-EFI Initialization) - Initialize memory
3. DXE (Driver Execution Environment) - Load drivers
4. BDS (Boot Device Selection) - Find boot device
5. Load EFI bootloader from ESP (EFI System Partition)
6. Bootloader loads kernel
```

### 1.3 Master Boot Record (MBR)

Structure of MBR (512 bytes total):

```
┌─────────────────────────────────────┐
│ Bootstrap Code (446 bytes)          │  ← Stage 1 bootloader
├─────────────────────────────────────┤
│ Partition Entry 1 (16 bytes)        │
├─────────────────────────────────────┤
│ Partition Entry 2 (16 bytes)        │  ← Partition table
├─────────────────────────────────────┤
│ Partition Entry 3 (16 bytes)        │
├─────────────────────────────────────┤
│ Partition Entry 4 (16 bytes)        │
├─────────────────────────────────────┤
│ Boot Signature (2 bytes: 0x55AA)    │  ← Magic number
└─────────────────────────────────────┘
```

Each partition entry contains:
- Boot flag (1 byte) - 0x80 = bootable
- CHS address of first sector (3 bytes)
- Partition type (1 byte)
- CHS address of last sector (3 bytes)
- LBA of first sector (4 bytes)
- Number of sectors (4 bytes)

### 1.4 GPT (GUID Partition Table)

```
┌─────────────────────────────────────┐
│ Protective MBR (LBA 0)              │  ← For backward compatibility
├─────────────────────────────────────┤
│ Primary GPT Header (LBA 1)          │
├─────────────────────────────────────┤
│ Partition Entries (LBA 2-33)        │  ← Up to 128 partitions
├─────────────────────────────────────┤
│                                     │
│ Actual Partitions                   │
│                                     │
├─────────────────────────────────────┤
│ Backup Partition Entries            │
├─────────────────────────────────────┤
│ Backup GPT Header (Last LBA)        │
└─────────────────────────────────────┘
```

### 1.5 GRUB (GRand Unified Bootloader)

GRUB is the most common Linux bootloader. It has three stages:

#### Stage 1 (boot.img)
- Size: 446 bytes (fits in MBR)
- Purpose: Load Stage 1.5 or Stage 2
- Location: First sector of disk

#### Stage 1.5 (core.img)
- Size: ~32KB
- Purpose: Understand filesystem to load Stage 2
- Location: MBR gap (between MBR and first partition)

#### Stage 2
- Full GRUB with all modules
- Location: /boot/grub/
- Displays boot menu
- Loads kernel and initramfs

```bash
# GRUB configuration file location
/boot/grub/grub.cfg          # Generated config
/etc/default/grub            # User settings
/etc/grub.d/                 # Scripts to generate config

# Regenerate GRUB config
sudo grub-mkconfig -o /boot/grub/grub.cfg

# Install GRUB to disk
sudo grub-install /dev/sda
```

#### GRUB Configuration Example

```bash
# /etc/default/grub
GRUB_DEFAULT=0                    # Default menu entry
GRUB_TIMEOUT=5                    # Seconds to wait
GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"  # Kernel parameters
GRUB_CMDLINE_LINUX=""             # Additional parameters
```

### 1.6 Kernel Loading Process

1. GRUB loads kernel image (`vmlinuz`) into memory
2. GRUB loads initial RAM filesystem (`initramfs` or `initrd`)
3. GRUB passes control to kernel with boot parameters
4. Kernel decompresses itself
5. Kernel initializes hardware
6. Kernel mounts initramfs as temporary root
7. initramfs loads necessary drivers
8. Kernel mounts real root filesystem
9. Kernel executes /sbin/init (or systemd)

```bash
# View kernel boot parameters
cat /proc/cmdline

# Example output:
# BOOT_IMAGE=/boot/vmlinuz-5.15.0-generic root=UUID=xxxx ro quiet splash
```

### 1.7 initramfs (Initial RAM Filesystem)

initramfs is a temporary root filesystem loaded into memory:

```bash
# Contents of initramfs
/bin/           # Essential binaries (busybox, etc.)
/sbin/          # System binaries
/etc/           # Configuration
/lib/           # Libraries and kernel modules
/lib/modules/   # Kernel modules for hardware
/init           # Main init script (PID 1 initially)
/scripts/       # Helper scripts

# Examine initramfs contents
lsinitramfs /boot/initrd.img-$(uname -r)

# Extract initramfs
unmkinitramfs /boot/initrd.img-$(uname -r) /tmp/initramfs/

# Regenerate initramfs
sudo update-initramfs -u
```

#### Why initramfs is needed:
1. Root filesystem might be on encrypted disk
2. Root filesystem might be on LVM/RAID
3. Root filesystem might need specific drivers
4. Kernel can't include all possible drivers

---

## 2. Linux Kernel

### 2.1 What is the Kernel?

The kernel is the core of the operating system. It's the bridge between hardware and software.

```
┌─────────────────────────────────────────────────────────┐
│                    User Applications                     │
├─────────────────────────────────────────────────────────┤
│                    System Libraries                      │
│                   (glibc, libpthread)                   │
├─────────────────────────────────────────────────────────┤
│                 System Call Interface                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│                    LINUX KERNEL                         │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐      │
│  │ Process │ │ Memory  │ │  File   │ │ Network │      │
│  │ Mgmt    │ │ Mgmt    │ │ Systems │ │ Stack   │      │
│  └─────────┘ └─────────┘ └─────────┘ └─────────┘      │
│  ┌─────────┐ ┌─────────┐ ┌─────────────────────┐      │
│  │  IPC    │ │ Device  │ │   Arch-Specific     │      │
│  │         │ │ Drivers │ │   Code (x86, ARM)   │      │
│  └─────────┘ └─────────┘ └─────────────────────┘      │
│                                                         │
├─────────────────────────────────────────────────────────┤
│                       Hardware                          │
│    CPU    Memory    Disk    Network    Peripherals      │
└─────────────────────────────────────────────────────────┘
```

### 2.2 Kernel Space vs User Space

```
┌─────────────────────────────────────────┐  High Memory
│                                         │  (0xFFFFFFFF on 32-bit)
│           KERNEL SPACE                  │
│   - Full hardware access                │
│   - Ring 0 (most privileged)            │
│   - Kernel code and data                │
│   - Device drivers                      │
│                                         │
├─────────────────────────────────────────┤  Kernel/User boundary
│                                         │
│            USER SPACE                   │
│   - Limited access via syscalls         │
│   - Ring 3 (least privileged)           │
│   - Applications                        │
│   - Libraries                           │
│                                         │
└─────────────────────────────────────────┘  Low Memory (0x00000000)
```

#### CPU Protection Rings (x86)

```
Ring 0 (Kernel Mode):
- Full access to all instructions
- Direct hardware access
- Memory management
- Interrupt handling

Ring 3 (User Mode):
- Limited instruction set
- Cannot access hardware directly
- Cannot access kernel memory
- Must use system calls
```

### 2.3 System Calls

System calls are the interface between user space and kernel space.

```c
// Example: How read() system call works

User Space:
    1. Application calls read(fd, buffer, count)
    2. glibc wrapper prepares arguments
    3. Triggers software interrupt (syscall instruction)

    ↓ Mode switch (User → Kernel)

Kernel Space:
    4. Kernel saves user context
    5. Looks up syscall number in syscall table
    6. Executes sys_read() kernel function
    7. Kernel restores user context

    ↓ Mode switch (Kernel → User)

User Space:
    8. Returns result to application
```

```bash
# View available system calls
man syscalls

# Trace system calls of a process
strace ls -la

# Common system calls:
# - read(), write()     - I/O operations
# - open(), close()     - File operations
# - fork(), exec()      - Process creation
# - mmap(), munmap()    - Memory mapping
# - socket(), connect() - Network operations
```

### 2.4 Kernel Modules

Kernel modules are pieces of code that can be loaded/unloaded at runtime.

```bash
# List loaded modules
lsmod

# Load a module
sudo modprobe module_name

# Remove a module
sudo modprobe -r module_name

# Show module info
modinfo module_name

# Module location
/lib/modules/$(uname -r)/

# Module dependencies
/lib/modules/$(uname -r)/modules.dep
```

#### Example: Writing a Simple Kernel Module

```c
// hello.c
#include <linux/init.h>
#include <linux/module.h>
#include <linux/kernel.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Your Name");
MODULE_DESCRIPTION("A simple Hello World module");
MODULE_VERSION("1.0");

static int __init hello_init(void) {
    printk(KERN_INFO "Hello, Kernel!\n");
    return 0;  // 0 = success
}

static void __exit hello_exit(void) {
    printk(KERN_INFO "Goodbye, Kernel!\n");
}

module_init(hello_init);
module_exit(hello_exit);
```

```makefile
# Makefile
obj-m += hello.o

all:
    make -C /lib/modules/$(shell uname -r)/build M=$(PWD) modules

clean:
    make -C /lib/modules/$(shell uname -r)/build M=$(PWD) clean
```

### 2.5 Kernel Data Structures

#### Task Struct (Process Descriptor)

```c
// Simplified task_struct
struct task_struct {
    volatile long state;           // Process state
    pid_t pid;                     // Process ID
    pid_t tgid;                    // Thread Group ID
    
    struct task_struct *parent;    // Parent process
    struct list_head children;     // Child processes
    
    struct mm_struct *mm;          // Memory descriptor
    struct files_struct *files;    // Open files
    struct fs_struct *fs;          // Filesystem info
    
    unsigned int policy;           // Scheduling policy
    int prio;                      // Priority
    
    struct cred *cred;            // Credentials
    char comm[TASK_COMM_LEN];     // Executable name
    // ... many more fields
};
```

### 2.6 Virtual File Systems in /proc and /sys

```bash
# /proc - Process and kernel information
/proc/cpuinfo          # CPU information
/proc/meminfo          # Memory information
/proc/[pid]/           # Per-process information
/proc/[pid]/status     # Process status
/proc/[pid]/maps       # Memory mappings
/proc/[pid]/fd/        # Open file descriptors
/proc/sys/             # Kernel parameters (tunable)

# /sys - Device and driver information
/sys/class/            # Device classes
/sys/block/            # Block devices
/sys/devices/          # Device hierarchy
/sys/module/           # Loaded modules
/sys/fs/               # Filesystem information
```

### 2.7 Kernel Parameters

```bash
# View all kernel parameters
sysctl -a

# View specific parameter
sysctl net.ipv4.ip_forward

# Set parameter temporarily
sudo sysctl -w net.ipv4.ip_forward=1

# Set parameter permanently
echo "net.ipv4.ip_forward = 1" | sudo tee /etc/sysctl.d/99-custom.conf
sudo sysctl -p /etc/sysctl.d/99-custom.conf

# Common parameters:
# vm.swappiness              - Swap usage tendency (0-100)
# net.core.somaxconn         - Max socket connections
# fs.file-max                - Max open files system-wide
# kernel.pid_max             - Max PIDs
```

---

## 3. Processes & Threads

### 3.1 What is a Process?

A process is an instance of a running program. It includes:
- Program code (text segment)
- Current activity (program counter, registers)
- Stack (temporary data)
- Data section (global variables)
- Heap (dynamically allocated memory)

```
Process Memory Layout:

┌─────────────────────────┐ High Address (0xFFFFFFFF)
│        Stack            │ ↓ Grows downward
│   (local variables,     │
│    function calls)      │
├─────────────────────────┤
│          ↓              │
│    (unused space)       │
│          ↑              │
├─────────────────────────┤
│         Heap            │ ↑ Grows upward
│   (dynamic allocation)  │
├─────────────────────────┤
│         BSS             │ Uninitialized global data
├─────────────────────────┤
│        Data             │ Initialized global data
├─────────────────────────┤
│        Text             │ Program code (read-only)
└─────────────────────────┘ Low Address (0x00000000)
```

### 3.2 Process States

```
                    ┌─────────────────┐
                    │     Created     │
                    │    (New)        │
                    └────────┬────────┘
                             │ admitted
                             ▼
         ┌───────────────────────────────────────┐
         │                                       │
         │  ┌─────────────┐    ┌─────────────┐  │
 exit    │  │             │    │             │  │ interrupt
 ────────┼──│   Running   │←───│    Ready    │←─┼─────────
         │  │             │    │             │  │
         │  └──────┬──────┘    └──────▲──────┘  │
         │         │                  │         │
         │         │ I/O or event     │         │
         │         │ wait             │ I/O or  │
         │         ▼                  │ event   │
         │  ┌─────────────┐          │ done    │
         │  │             │          │         │
         │  │   Waiting   │──────────┘         │
         │  │  (Blocked)  │                    │
         │  └─────────────┘                    │
         │                                     │
         └─────────────────────────────────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │   Terminated    │
                    │    (Zombie)     │
                    └─────────────────┘
```

Linux Process States:
```bash
# State codes in ps output:
R - Running or runnable
S - Interruptible sleep (waiting for event)
D - Uninterruptible sleep (usually I/O)
T - Stopped (by signal or debugger)
Z - Zombie (terminated but not reaped)
X - Dead
```

### 3.3 Process Creation

```c
// fork() - Create child process
#include <stdio.h>
#include <unistd.h>
#include <sys/wait.h>

int main() {
    pid_t pid = fork();
    
    if (pid < 0) {
        // Error
        perror("fork failed");
        return 1;
    } else if (pid == 0) {
        // Child process
        printf("Child PID: %d, Parent PID: %d\n", getpid(), getppid());
        // Often followed by exec()
        execlp("ls", "ls", "-la", NULL);
    } else {
        // Parent process
        printf("Parent PID: %d, Child PID: %d\n", getpid(), pid);
        wait(NULL);  // Wait for child to complete
    }
    
    return 0;
}
```

#### Copy-on-Write (COW)

When `fork()` is called:
1. Child gets copy of parent's page table (not actual memory)
2. Both processes share same physical pages (marked read-only)
3. When either writes to a page, kernel creates a copy
4. This makes fork() very efficient

```
Before write:
Parent:  [Page Table] → Page 1 (shared) → Physical Memory
Child:   [Page Table] → Page 1 (shared) ↗

After write by child:
Parent:  [Page Table] → Page 1 → Physical Memory (original)
Child:   [Page Table] → Page 1 → Physical Memory (copy)
```

### 3.4 Process vs Thread

```
┌───────────────────────────────────────────────────────────┐
│                        PROCESS                            │
│  ┌─────────────────────────────────────────────────────┐ │
│  │                   Address Space                      │ │
│  │  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐   │ │
│  │  │  Code   │ │  Data   │ │  Heap   │ │  Files  │   │ │
│  │  │(shared) │ │(shared) │ │(shared) │ │(shared) │   │ │
│  │  └─────────┘ └─────────┘ └─────────┘ └─────────┘   │ │
│  │                                                      │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐ │ │
│  │  │  Thread 1   │  │  Thread 2   │  │  Thread 3   │ │ │
│  │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │ │ │
│  │  │ │  Stack  │ │  │ │  Stack  │ │  │ │  Stack  │ │ │ │
│  │  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │ │ │
│  │  │ ┌─────────┐ │  │ ┌─────────┐ │  │ ┌─────────┐ │ │ │
│  │  │ │Registers│ │  │ │Registers│ │  │ │Registers│ │ │ │
│  │  │ └─────────┘ │  │ └─────────┘ │  │ └─────────┘ │ │ │
│  │  │     PC      │  │     PC      │  │     PC      │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘ │ │
│  └─────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────┘

Threads share: Code, Data, Heap, Files, Signal handlers
Threads have own: Stack, Registers, Program Counter, Thread ID
```

### 3.5 Thread Implementation in Linux

Linux implements threads using `clone()` system call:

```c
// POSIX Threads (pthreads)
#include <pthread.h>
#include <stdio.h>

void* thread_function(void* arg) {
    int* num = (int*)arg;
    printf("Thread received: %d\n", *num);
    return NULL;
}

int main() {
    pthread_t thread;
    int arg = 42;
    
    // Create thread
    pthread_create(&thread, NULL, thread_function, &arg);
    
    // Wait for thread to complete
    pthread_join(thread, NULL);
    
    return 0;
}

// Compile: gcc -pthread program.c -o program
```

### 3.6 Process Scheduling

#### Scheduling Classes in Linux

```
┌─────────────────────────────────────────────────────────┐
│                  SCHEDULING CLASSES                     │
├─────────────────────────────────────────────────────────┤
│  SCHED_DEADLINE  │ Earliest Deadline First            │
│  (Highest)       │ Real-time, deadline-based          │
├─────────────────────────────────────────────────────────┤
│  SCHED_FIFO      │ First In, First Out                │
│  SCHED_RR        │ Round Robin                        │
│  (Real-time)     │ Priority: 1-99                     │
├─────────────────────────────────────────────────────────┤
│  SCHED_OTHER     │ Completely Fair Scheduler (CFS)    │
│  SCHED_BATCH     │ For batch jobs                     │
│  SCHED_IDLE      │ Very low priority                  │
│  (Normal)        │ Nice: -20 to 19                    │
└─────────────────────────────────────────────────────────┘
```

#### Completely Fair Scheduler (CFS)

CFS is the default scheduler for normal processes:

```
Virtual Runtime = Actual Runtime × (Weight of nice 0 / Weight of process)

Nice Value    Weight      Relative Time
-20           88761       10x more CPU
-10           9548        3x more CPU
  0           1024        baseline
 10           110         ~1/10 CPU
 19           15          very little CPU

Red-Black Tree (Ordered by vruntime):

            [P3: vruntime=50]
           /                \
    [P1: vruntime=30]  [P5: vruntime=80]
    /            \
[P2: vruntime=10] [P4: vruntime=40]
       ↑
  Next to run (leftmost = smallest vruntime)
```

```bash
# View process priority
ps -eo pid,ni,pri,comm

# Change nice value
nice -n 10 command           # Start with nice 10
renice -n 5 -p PID           # Change running process

# Set real-time priority
chrt -f -p 50 PID            # FIFO with priority 50
chrt -r -p 50 PID            # Round Robin with priority 50
```

### 3.7 Process Communication (IPC)

#### Types of IPC

```
1. Pipes (Anonymous)
   Parent ──────[pipe]────── Child
   - Unidirectional
   - Related processes only

2. Named Pipes (FIFOs)
   Process A ──────[/tmp/myfifo]────── Process B
   - Unidirectional
   - Unrelated processes
   - Persistent in filesystem

3. Message Queues
   ┌────────────────────────────┐
   │ Message Queue              │
   │ [Msg1][Msg2][Msg3]...      │
   └────────────────────────────┘
   - Multiple senders/receivers
   - Message boundaries preserved

4. Shared Memory
   ┌─────────────────────────────┐
   │     Shared Memory Segment   │
   │         (fastest IPC)       │
   └─────────────────────────────┘
        ↑                ↑
   Process A        Process B
   
5. Semaphores
   - Synchronization primitive
   - Controls access to shared resources

6. Sockets
   - Local (Unix domain) or network
   - Bidirectional
   - Client-server model

7. Signals
   - Asynchronous notifications
   - Limited information (just signal number)
```

```bash
# View IPC resources
ipcs                    # All IPC resources
ipcs -m                 # Shared memory
ipcs -q                 # Message queues
ipcs -s                 # Semaphores

# Remove IPC resources
ipcrm -m <shmid>        # Remove shared memory
ipcrm -q <msqid>        # Remove message queue
```

### 3.8 Signals

```bash
# Common signals
Signal      Number  Default Action   Description
───────────────────────────────────────────────────
SIGHUP      1       Terminate        Hangup
SIGINT      2       Terminate        Interrupt (Ctrl+C)
SIGQUIT     3       Core dump        Quit (Ctrl+\)
SIGKILL     9       Terminate        Kill (cannot be caught)
SIGSEGV     11      Core dump        Segmentation fault
SIGTERM     15      Terminate        Termination request
SIGSTOP     19      Stop             Stop (cannot be caught)
SIGCONT     18      Continue         Continue if stopped
SIGCHLD     17      Ignore           Child terminated

# Send signals
kill -SIGTERM PID      # Graceful termination
kill -9 PID            # Force kill
kill -STOP PID         # Pause process
kill -CONT PID         # Resume process
killall process_name   # Kill by name
pkill pattern          # Kill by pattern
```

### 3.9 Process Monitoring

```bash
# ps - Process status
ps aux                     # All processes, detailed
ps -ef                     # Full format listing
ps -eo pid,ppid,cmd,%cpu,%mem  # Custom columns
ps --forest                # Process tree

# top - Dynamic view
top                        # Interactive process viewer
top -p PID                 # Monitor specific process

# htop - Enhanced top
htop                       # Better interface

# Process tree
pstree                     # Show process hierarchy
pstree -p                  # With PIDs

# /proc filesystem
cat /proc/PID/status       # Process status
cat /proc/PID/cmdline      # Command line
ls -la /proc/PID/fd        # Open files
cat /proc/PID/maps         # Memory mappings
cat /proc/PID/limits       # Resource limits
```

---

## 4. Systemd & Init System

### 4.1 What is an Init System?

The init system is the first process started by the kernel (PID 1). It:
- Starts all other processes
- Manages system services
- Handles orphaned processes
- Manages system state

#### Evolution of Init Systems

```
SysVinit (1983) → Upstart (2006) → systemd (2010)
    ↓                  ↓               ↓
Sequential        Event-based     Parallel + 
scripts           parallel        Dependencies
```

### 4.2 Systemd Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        systemd                               │
├─────────────────────────────────────────────────────────────┤
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐   │
│  │  systemd  │ │ systemd-  │ │ systemd-  │ │ systemd-  │   │
│  │  (PID 1)  │ │ journald  │ │ logind    │ │ networkd  │   │
│  │           │ │ (logging) │ │ (login)   │ │ (network) │   │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘   │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐   │
│  │ systemd-  │ │ systemd-  │ │ systemd-  │ │ systemd-  │   │
│  │ udevd     │ │ resolved  │ │ timesyncd │ │ hostnamed │   │
│  │ (devices) │ │ (DNS)     │ │ (NTP)     │ │ (hostname)│   │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘   │
├─────────────────────────────────────────────────────────────┤
│                         D-Bus                                │
│                  (Inter-process messaging)                   │
└─────────────────────────────────────────────────────────────┘
```

### 4.3 Systemd Units

Units are the basic building blocks of systemd:

```
Unit Type      Extension      Purpose
──────────────────────────────────────────────────────
Service        .service       System services (daemons)
Socket         .socket        Socket-based activation
Target         .target        Group of units (like runlevel)
Mount          .mount         Filesystem mount points
Automount      .automount     On-demand mounting
Timer          .timer         Timer-based activation (cron-like)
Path           .path          File/directory monitoring
Device         .device        Kernel device
Swap           .swap          Swap space
Slice          .slice         Resource management (cgroups)
Scope          .scope         Externally created processes
```

### 4.4 Unit File Locations

```bash
/etc/systemd/system/          # Local configuration (highest priority)
/run/systemd/system/          # Runtime units
/lib/systemd/system/          # Distribution-provided units

# User units
~/.config/systemd/user/       # User-specific units
/etc/systemd/user/            # System-wide user units
```

### 4.5 Service Unit File Structure

```ini
# /etc/systemd/system/myservice.service

[Unit]
Description=My Custom Service
Documentation=https://example.com/docs
After=network.target           # Start after network
Wants=network.target           # Weak dependency
Requires=mysql.service         # Strong dependency
Before=other.service           # Start before other

[Service]
Type=simple                    # Service type
User=myuser                    # Run as user
Group=mygroup                  # Run as group
WorkingDirectory=/opt/myapp    # Working directory
Environment=NODE_ENV=production # Environment variables
EnvironmentFile=/etc/myapp/env # Environment file
ExecStartPre=/usr/bin/pre-script  # Run before starting
ExecStart=/usr/bin/myapp       # Main command
ExecStartPost=/usr/bin/post-script # Run after starting
ExecReload=/bin/kill -HUP $MAINPID # Reload command
ExecStop=/bin/kill -TERM $MAINPID  # Stop command
Restart=on-failure             # Restart policy
RestartSec=5                   # Delay before restart
TimeoutStartSec=30             # Startup timeout
TimeoutStopSec=30              # Shutdown timeout

# Resource limits
LimitNOFILE=65536             # Max open files
LimitNPROC=4096               # Max processes
MemoryLimit=1G                # Memory limit

# Security
PrivateTmp=true               # Private /tmp
ProtectSystem=full            # Read-only /usr, /boot
ProtectHome=true              # Hide /home
NoNewPrivileges=true          # Prevent privilege escalation

[Install]
WantedBy=multi-user.target    # Enable for multi-user
```

#### Service Types

```
simple     - ExecStart process is the main process (default)
forking    - ExecStart forks and parent exits
oneshot    - Process exits after completion
dbus       - Ready when D-Bus name acquired
notify     - Ready when sends notification
idle       - Run when all jobs dispatched
```

### 4.6 systemctl Commands

```bash
# Service management
systemctl start service        # Start a service
systemctl stop service         # Stop a service
systemctl restart service      # Restart a service
systemctl reload service       # Reload configuration
systemctl status service       # Check status
systemctl enable service       # Enable at boot
systemctl disable service      # Disable at boot
systemctl is-active service    # Check if running
systemctl is-enabled service   # Check if enabled

# List units
systemctl list-units           # Active units
systemctl list-units --all     # All units
systemctl list-unit-files      # All installed
systemctl list-dependencies    # Show dependencies

# System state
systemctl get-default          # Current default target
systemctl set-default target   # Set default target
systemctl isolate target       # Switch to target
systemctl rescue               # Enter rescue mode
systemctl emergency            # Enter emergency mode

# System control
systemctl poweroff             # Shut down
systemctl reboot               # Reboot
systemctl suspend              # Suspend
systemctl hibernate            # Hibernate

# Reload systemd
systemctl daemon-reload        # Reload unit files
```

### 4.7 Targets (Runlevels)

```
Target              SysV Runlevel    Description
────────────────────────────────────────────────────────
poweroff.target     0                Halt system
rescue.target       1                Single user mode
multi-user.target   3                Multi-user, no GUI
graphical.target    5                Multi-user with GUI
reboot.target       6                Reboot

# Target dependencies (simplified)
graphical.target
    └── multi-user.target
            └── basic.target
                    └── sysinit.target
                            └── local-fs.target
                                    └── -.mount (root)
```

```bash
# Change target
systemctl isolate multi-user.target    # Switch to multi-user
systemctl isolate graphical.target     # Switch to graphical
```

### 4.8 Journald (Logging)

```bash
# View logs
journalctl                     # All logs
journalctl -b                  # Current boot
journalctl -b -1               # Previous boot
journalctl -f                  # Follow (like tail -f)
journalctl -u service          # Specific service
journalctl -p err              # Priority error and above
journalctl --since "1 hour ago"  # Time-based
journalctl --since "2024-01-01" --until "2024-01-02"

# Output formats
journalctl -o verbose          # Detailed output
journalctl -o json             # JSON format
journalctl -o json-pretty      # Pretty JSON

# Log priorities
0 - emerg      # System unusable
1 - alert      # Immediate action needed
2 - crit       # Critical conditions
3 - err        # Error conditions
4 - warning    # Warning conditions
5 - notice     # Normal but significant
6 - info       # Informational
7 - debug      # Debug messages

# Configuration
/etc/systemd/journald.conf
```

### 4.9 Timers (Cron Alternative)

```ini
# /etc/systemd/system/backup.timer
[Unit]
Description=Daily Backup Timer

[Timer]
OnCalendar=daily               # Run daily at midnight
# OnCalendar=*-*-* 02:00:00    # Every day at 2 AM
# OnBootSec=10min              # 10 minutes after boot
# OnUnitActiveSec=1h           # 1 hour after last activation
Persistent=true                # Catch up missed runs

[Install]
WantedBy=timers.target
```

```bash
# List timers
systemctl list-timers

# Enable timer
systemctl enable backup.timer
systemctl start backup.timer
```

### 4.10 Cgroups (Control Groups)

systemd uses cgroups for resource management:

```bash
# View cgroup hierarchy
systemd-cgls

# Resource accounting
systemd-cgtop                  # Top for cgroups

# View unit cgroup
systemctl show service --property=ControlGroup

# Set resource limits in unit file
[Service]
CPUQuota=50%                   # Max 50% CPU
MemoryMax=500M                 # Max 500MB memory
IOWeight=100                   # I/O priority (1-10000)
TasksMax=100                   # Max tasks/threads
```

---

## 5. File System

### 5.1 Linux File System Hierarchy

```
/                              Root of entire filesystem
├── bin/                       Essential user binaries
├── boot/                      Boot loader files, kernel
│   ├── grub/                  GRUB bootloader
│   ├── vmlinuz-*              Kernel image
│   └── initrd.img-*           Initial RAM disk
├── dev/                       Device files
│   ├── sda                    First SATA disk
│   ├── sda1                   First partition
│   ├── null                   Null device
│   ├── zero                   Zero device
│   ├── random                 Random generator
│   └── tty*                   Terminals
├── etc/                       System configuration
│   ├── passwd                 User accounts
│   ├── shadow                 Password hashes
│   ├── group                  Group definitions
│   ├── fstab                  Filesystem mount table
│   ├── hosts                  Hostname resolution
│   └── systemd/               Systemd configuration
├── home/                      User home directories
├── lib/                       Essential shared libraries
├── lib64/                     64-bit libraries
├── media/                     Removable media mount points
├── mnt/                       Temporary mount points
├── opt/                       Optional software
├── proc/                      Virtual filesystem (processes)
├── root/                      Root user's home directory
├── run/                       Runtime variable data
├── sbin/                      System binaries
├── srv/                       Service data
├── sys/                       Virtual filesystem (kernel/hardware)
├── tmp/                       Temporary files
├── usr/                       User programs and data
│   ├── bin/                   User binaries
│   ├── lib/                   Libraries
│   ├── local/                 Locally installed software
│   ├── sbin/                  Non-essential system binaries
│   └── share/                 Architecture-independent data
└── var/                       Variable data
    ├── log/                   Log files
    ├── cache/                 Application cache
    ├── lib/                   Variable state data
    ├── spool/                 Spool directories
    └── tmp/                   Temporary files preserved across reboot
```

### 5.2 File Types in Linux

```bash
# File types (ls -l first character)
-    Regular file
d    Directory
l    Symbolic link
c    Character device (e.g., /dev/tty)
b    Block device (e.g., /dev/sda)
p    Named pipe (FIFO)
s    Socket

# Identify file type
file /path/to/file
stat /path/to/file
```

### 5.3 Inodes and Data Blocks

```
Filesystem Structure:
┌─────────────────────────────────────────────────────────┐
│ Boot Block │ Super Block │ Inode Table │ Data Blocks   │
└─────────────────────────────────────────────────────────┘

Inode Structure:
┌────────────────────────────────────────┐
│ Inode Number: 12345                    │
├────────────────────────────────────────┤
│ Mode (permissions): -rwxr-xr-x         │
│ Owner UID: 1000                        │
│ Group GID: 1000                        │
│ Size: 4096 bytes                       │
│ Timestamps:                            │
│   - atime (access time)                │
│   - mtime (modification time)          │
│   - ctime (inode change time)          │
│ Link count: 2                          │
├────────────────────────────────────────┤
│ Direct pointers (12):                  │
│   [ptr1][ptr2]...[ptr12] → Data blocks │
├────────────────────────────────────────┤
│ Single indirect pointer:               │
│   [ptr] → [pointer block] → Data       │
├────────────────────────────────────────┤
│ Double indirect pointer:               │
│   [ptr] → [ptrs] → [ptrs] → Data       │
├────────────────────────────────────────┤
│ Triple indirect pointer:               │
│   [ptr] → [ptrs] → [ptrs] → [ptrs] → Data │
└────────────────────────────────────────┘
```

```bash
# View inode information
ls -i file                    # Show inode number
stat file                     # Detailed inode info
df -i                         # Inode usage per filesystem

# Find by inode
find / -inum 12345            # Find file by inode number
```

### 5.4 Hard Links vs Symbolic Links

```
Hard Link:
┌──────────┐     ┌──────────┐
│ file1    │────→│  Inode   │────→ Data blocks
└──────────┘     │ (12345)  │
┌──────────┐     │ Links: 2 │
│ file2    │────→│          │
└──────────┘     └──────────┘

- Same inode number
- Same file data
- Link count increases
- Cannot span filesystems
- Cannot link directories
- Deleting one doesn't affect others

Symbolic Link:
┌──────────┐     ┌──────────┐
│ link     │────→│  Inode   │────→ "/path/to/target"
└──────────┘     │ (67890)  │      (stored as data)
                 └──────────┘
                       │
                       ↓
┌──────────┐     ┌──────────┐
│ target   │────→│  Inode   │────→ Actual data
└──────────┘     │ (12345)  │
                 └──────────┘

- Different inode numbers
- Link contains path to target
- Can span filesystems
- Can link directories
- Breaks if target deleted
```

```bash
# Create hard link
ln original hardlink

# Create symbolic link
ln -s /path/to/target symlink

# View link information
ls -la                        # Shows symlink target
readlink symlink              # Show symlink target
readlink -f symlink           # Show absolute path
```

### 5.5 Common Filesystems

```
Filesystem     Description                  Max File Size  Max Volume
──────────────────────────────────────────────────────────────────────
ext4           Default Linux filesystem     16 TB          1 EB
XFS            High-performance, scalable   8 EB           8 EB
Btrfs          Copy-on-write, snapshots     16 EB          16 EB
ZFS            Advanced features            16 EB          256 trillion YB
NTFS           Windows filesystem           16 EB          256 TB
FAT32          Universal, limited           4 GB           2 TB
exFAT          Extended FAT                 128 PB         128 PB
```

### 5.6 Mounting Filesystems

```bash
# Mount a filesystem
mount /dev/sdb1 /mnt/data
mount -t ext4 /dev/sdb1 /mnt/data
mount -o ro /dev/sdb1 /mnt/data          # Read-only

# Mount options
mount -o rw,noexec,nosuid /dev/sdb1 /mnt/data

Common options:
  rw        - Read-write
  ro        - Read-only
  noexec    - Prevent execution
  nosuid    - Ignore SUID bits
  nodev     - Ignore device files
  noatime   - Don't update access time
  sync      - Synchronous I/O

# Unmount
umount /mnt/data
umount -l /mnt/data              # Lazy unmount

# View mounts
mount                            # All mounts
findmnt                          # Tree view
cat /proc/mounts                 # Kernel's view
df -h                            # Disk usage
```

### 5.7 /etc/fstab (Filesystem Table)

```bash
# /etc/fstab format:
# <device>      <mount point>  <type>  <options>       <dump> <pass>
/dev/sda1       /              ext4    defaults        0      1
/dev/sda2       /home          ext4    defaults        0      2
UUID=xxx-xxx    /data          ext4    defaults,noatime 0     2
/dev/sda3       none           swap    sw              0      0
tmpfs           /tmp           tmpfs   defaults,size=2G 0     0
/dev/sr0        /media/cdrom   auto    ro,noauto,user  0      0

# Fields:
# dump (5th): 0 = no backup, 1 = backup with dump
# pass (6th): 0 = no fsck, 1 = root (first), 2 = other filesystems
```

```bash
# Get UUID
blkid /dev/sda1

# Reload fstab
mount -a                         # Mount all in fstab
systemctl daemon-reload          # If using systemd mount units
```

### 5.8 Disk Management

```bash
# Disk information
lsblk                            # List block devices
lsblk -f                         # With filesystem info
fdisk -l                         # Partition tables
parted -l                        # Partition info

# Partition management
fdisk /dev/sdb                   # MBR partitioning
gdisk /dev/sdb                   # GPT partitioning
parted /dev/sdb                  # Both MBR and GPT

# Create filesystem
mkfs.ext4 /dev/sdb1              # ext4 filesystem
mkfs.xfs /dev/sdb1               # XFS filesystem
mkswap /dev/sdb2                 # Swap space

# Filesystem check
fsck /dev/sdb1                   # Check filesystem
fsck -y /dev/sdb1                # Auto-fix
e2fsck -f /dev/sdb1              # Force check ext4

# Disk usage
du -sh /path                     # Directory size
du -h --max-depth=1              # Subdirectory sizes
ncdu /path                       # Interactive viewer
```

### 5.9 LVM (Logical Volume Manager)

```
Physical Disks → Physical Volumes → Volume Group → Logical Volumes → Filesystems

┌─────────────────────────────────────────────────────────────────┐
│                        Volume Group (vg1)                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │     LV1      │  │     LV2      │  │     LV3      │          │
│  │   (root)     │  │   (home)     │  │   (data)     │          │
│  │    20GB      │  │    50GB      │  │    30GB      │          │
│  └──────────────┘  └──────────────┘  └──────────────┘          │
│                                                                  │
├──────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────┐  ┌─────────────────────────┐       │
│  │    PV: /dev/sda1        │  │    PV: /dev/sdb1        │       │
│  │        50GB             │  │        50GB             │       │
│  └─────────────────────────┘  └─────────────────────────┘       │
└──────────────────────────────────────────────────────────────────┘
```

```bash
# Create physical volume
pvcreate /dev/sdb1

# Create volume group
vgcreate vg1 /dev/sdb1 /dev/sdc1

# Create logical volume
lvcreate -n lv_data -L 50G vg1

# Extend logical volume
lvextend -L +10G /dev/vg1/lv_data
resize2fs /dev/vg1/lv_data        # Resize ext4

# View LVM info
pvdisplay                          # Physical volumes
vgdisplay                          # Volume groups
lvdisplay                          # Logical volumes
```

### 5.10 File Permissions

```bash
Permission Structure:
┌─────┬─────┬─────┬─────┐
│Type │User │Group│Other│
├─────┼─────┼─────┼─────┤
│  -  │ rwx │ r-x │ r-- │
└─────┴─────┴─────┴─────┘

Type: - (file), d (dir), l (link), etc.

Permission   Octal   Meaning (File)      Meaning (Directory)
─────────────────────────────────────────────────────────────
r (read)     4       Read contents       List contents
w (write)    2       Modify contents     Create/delete files
x (execute)  1       Execute file        Enter directory

# Change permissions
chmod 755 file                    # rwxr-xr-x
chmod u+x file                    # Add execute for user
chmod g-w file                    # Remove write for group
chmod o=r file                    # Set other to read only
chmod -R 755 directory            # Recursive

# Change ownership
chown user file                   # Change owner
chown user:group file             # Change owner and group
chown -R user:group directory     # Recursive
chgrp group file                  # Change group only
```

### 5.11 Special Permissions

```bash
Special Bits:
┌─────────────────────────────────────────────────────────┐
│ SUID (4)     │ Execute as file owner                   │
│ SGID (2)     │ Execute as file group (or inherit dir)  │
│ Sticky (1)   │ Only owner can delete (in directory)    │
└─────────────────────────────────────────────────────────┘

# SUID - Set User ID (runs as file owner)
chmod u+s file                    # Add SUID
chmod 4755 file                   # SUID + rwxr-xr-x
ls -l /usr/bin/passwd             # -rwsr-xr-x (s = SUID)

# SGID - Set Group ID
chmod g+s file                    # Add SGID
chmod 2755 directory              # Files inherit group
ls -l                             # -rwxr-sr-x (s = SGID)

# Sticky bit (usually on /tmp)
chmod +t directory                # Add sticky
chmod 1777 /tmp                   # Sticky + rwxrwxrwx
ls -ld /tmp                       # drwxrwxrwt (t = sticky)
```

### 5.12 Access Control Lists (ACLs)

```bash
# View ACLs
getfacl file

# Set ACLs
setfacl -m u:john:rwx file        # User john gets rwx
setfacl -m g:devs:rx file         # Group devs gets rx
setfacl -m o::r file              # Others get r

# Default ACLs (for directories)
setfacl -d -m u:john:rwx directory  # New files inherit

# Remove ACLs
setfacl -x u:john file            # Remove user entry
setfacl -b file                   # Remove all ACLs

# Example output:
# file: test.txt
# owner: alice
# group: staff
# user::rw-
# user:john:rwx
# group::r--
# group:devs:rx
# mask::rwx
# other::r--
```

---

## 6. User Management

### 6.1 User and Group Concepts

```
User Account Components:
┌─────────────────────────────────────────────────────────┐
│ Username      │ Human-readable identifier              │
│ UID           │ Numeric user ID                        │
│ GID           │ Primary group ID                       │
│ Home Dir      │ User's home directory                  │
│ Shell         │ Default login shell                    │
│ Password      │ Hashed in /etc/shadow                  │
│ GECOS         │ User info (name, phone, etc.)          │
└─────────────────────────────────────────────────────────┘

Special UIDs:
0       - root (superuser)
1-99    - System accounts (distro-specific)
100-999 - System accounts (dynamic)
1000+   - Regular users
65534   - nobody (unprivileged)
```

### 6.2 User Database Files

```bash
# /etc/passwd - User account information
# Format: username:x:UID:GID:GECOS:home:shell
root:x:0:0:root:/root:/bin/bash
john:x:1000:1000:John Doe:/home/john:/bin/bash
www-data:x:33:33:www-data:/var/www:/usr/sbin/nologin

# /etc/shadow - Password hashes (root only)
# Format: username:hash:lastchange:min:max:warn:inactive:expire:reserved
root:$6$xyz...:18500:0:99999:7:::
john:$6$abc...:18500:0:99999:7:::

Hash format: $algorithm$salt$hash
  $1$  = MD5 (deprecated)
  $5$  = SHA-256
  $6$  = SHA-512 (recommended)
  $y$  = yescrypt (newest)

# /etc/group - Group information
# Format: groupname:x:GID:members
root:x:0:
sudo:x:27:john,jane
developers:x:1001:john,bob,alice

# /etc/gshadow - Group passwords (rarely used)
# Format: groupname:password:admins:members
```

### 6.3 User Management Commands

```bash
# Add user
useradd username                  # Basic (no home dir)
useradd -m username               # Create home directory
useradd -m -s /bin/bash -G sudo,docker username  # Full options

# useradd options:
  -m              # Create home directory
  -d /path        # Specify home directory
  -s /bin/bash    # Specify shell
  -g group        # Primary group
  -G groups       # Additional groups (comma-separated)
  -u UID          # Specific UID
  -c "comment"    # GECOS field
  -e YYYY-MM-DD   # Expiration date
  -r              # System account (UID < 1000)

# Alternative: adduser (interactive, Debian/Ubuntu)
adduser username

# Modify user
usermod -aG docker username       # Add to group
usermod -s /bin/zsh username      # Change shell
usermod -L username               # Lock account
usermod -U username               # Unlock account
usermod -d /new/home username     # Change home dir
usermod -l newname oldname        # Rename user

# Delete user
userdel username                  # Delete user only
userdel -r username               # Delete with home dir

# Password management
passwd username                   # Set/change password
passwd -l username                # Lock password
passwd -u username                # Unlock password
passwd -e username                # Expire (force change)
passwd -S username                # Show status

# View user info
id username                       # UID, GID, groups
whoami                            # Current user
who                               # Logged in users
w                                 # Who + what they're doing
last                              # Login history
lastlog                           # Last login for all users
```

### 6.4 Group Management

```bash
# Create group
groupadd groupname
groupadd -g 2000 groupname        # Specific GID

# Modify group
groupmod -n newname oldname       # Rename
groupmod -g 2001 groupname        # Change GID

# Delete group
groupdel groupname

# Group membership
gpasswd -a user group             # Add user to group
gpasswd -d user group             # Remove from group
gpasswd -A user group             # Make user admin

# View groups
groups username                   # Show user's groups
getent group groupname            # Group info
```

### 6.5 Switching Users

```bash
# Switch user
su username                       # Switch (need password)
su - username                     # Switch with environment
su -                              # Switch to root

# Sudo - Execute as another user
sudo command                      # Run as root
sudo -u username command          # Run as specific user
sudo -i                           # Interactive root shell
sudo -s                           # Shell as root
sudo -l                           # List allowed commands
sudo -v                           # Extend timeout
sudo -k                           # Invalidate cached credentials
```

### 6.6 Sudoers Configuration

```bash
# Edit sudoers (always use visudo!)
sudo visudo

# /etc/sudoers format:
# user  host=(runas) commands

# Examples:
root    ALL=(ALL:ALL) ALL              # Root can do anything
%sudo   ALL=(ALL:ALL) ALL              # sudo group can do anything
john    ALL=(ALL) NOPASSWD: ALL        # No password needed
jane    ALL=(ALL) /usr/bin/apt         # Only apt command
bob     ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx

# Aliases
User_Alias ADMINS = john, jane, bob
Cmnd_Alias SERVICES = /usr/bin/systemctl start *, /usr/bin/systemctl stop *
ADMINS ALL=(ALL) SERVICES

# Include directory
@includedir /etc/sudoers.d

# Create custom file
sudo visudo -f /etc/sudoers.d/custom
```

### 6.7 User Resource Limits

```bash
# /etc/security/limits.conf
# Format: <domain> <type> <item> <value>

# Examples:
*               soft    nofile          65535    # All users, soft limit
*               hard    nofile          65535    # All users, hard limit
@developers     soft    nproc           2048     # Group limit
john            hard    as              4000000  # Max memory (KB)

# Items:
  nofile    - Max open files
  nproc     - Max processes
  stack     - Max stack size (KB)
  data      - Max data size (KB)
  fsize     - Max file size (KB)
  memlock   - Max locked memory (KB)
  cpu       - Max CPU time (minutes)
  as        - Address space limit (KB)

# View limits
ulimit -a                         # All limits
ulimit -n                         # Open files limit

# Set temporary limits
ulimit -n 65535                   # Set open files

# For systemd services, use unit files:
# [Service]
# LimitNOFILE=65535
```

### 6.8 PAM (Pluggable Authentication Modules)

```bash
# PAM configuration location
/etc/pam.d/                       # Service-specific configs
/etc/security/                    # PAM module configs

# PAM config format:
# type    control    module    arguments

# Example: /etc/pam.d/sshd
auth       required   pam_sepermit.so
auth       include    common-auth
account    required   pam_nologin.so
account    include    common-account
session    include    common-session
session    optional   pam_motd.so

# Control values:
  required    - Must pass, continue checking
  requisite   - Must pass, stop on failure
  sufficient  - If passes, skip remaining same type
  optional    - Result ignored unless only module

# Common modules:
  pam_unix.so      - Traditional authentication
  pam_ldap.so      - LDAP authentication
  pam_pwquality.so - Password quality checking
  pam_limits.so    - Resource limits
  pam_tally2.so    - Login attempt limiting
```

---

## 7. Security

### 7.1 Security Overview

```
Linux Security Layers:
┌─────────────────────────────────────────────────────────────────┐
│                     Application Security                        │
│         (Input validation, authentication, encryption)          │
├─────────────────────────────────────────────────────────────────┤
│                        Mandatory Access Control                 │
│              (SELinux, AppArmor, TOMOYO)                       │
├─────────────────────────────────────────────────────────────────┤
│                     Discretionary Access Control                │
│              (File permissions, ACLs, capabilities)             │
├─────────────────────────────────────────────────────────────────┤
│                        Network Security                         │
│         (iptables/nftables, firewalld, TCP wrappers)           │
├─────────────────────────────────────────────────────────────────┤
│                        Kernel Security                          │
│        (Namespaces, cgroups, seccomp, kernel hardening)        │
├─────────────────────────────────────────────────────────────────┤
│                        Hardware Security                        │
│            (TPM, Secure Boot, Hardware encryption)              │
└─────────────────────────────────────────────────────────────────┘
```

### 7.2 Firewall with iptables

```bash
# iptables chains
INPUT     - Incoming packets destined for local system
OUTPUT    - Outgoing packets from local system
FORWARD   - Packets routed through the system

# Basic syntax
iptables -A CHAIN -p protocol --dport port -j ACTION

# View rules
iptables -L -v -n                # List all rules
iptables -L -v -n --line-numbers # With line numbers

# Basic rules
iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -i lo -j ACCEPT                    # Allow loopback
iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT  # Allow ping
iptables -A INPUT -p tcp --dport 22 -j ACCEPT        # Allow SSH
iptables -A INPUT -p tcp --dport 80 -j ACCEPT        # Allow HTTP
iptables -A INPUT -p tcp --dport 443 -j ACCEPT       # Allow HTTPS
iptables -A INPUT -j DROP                            # Drop everything else

# Delete rule
iptables -D INPUT 3                # Delete rule #3
iptables -F                        # Flush all rules

# Save rules
iptables-save > /etc/iptables.rules
iptables-restore < /etc/iptables.rules

# NAT (Network Address Translation)
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE  # Source NAT
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

### 7.3 Firewall with nftables (Modern)

```bash
# nftables is the successor to iptables

# Basic configuration
nft list ruleset                  # View all rules

# Create table and chain
nft add table inet filter
nft add chain inet filter input { type filter hook input priority 0 \; policy drop \; }

# Add rules
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input iif lo accept
nft add rule inet filter input tcp dport 22 accept
nft add rule inet filter input tcp dport { 80, 443 } accept

# Save configuration
nft list ruleset > /etc/nftables.conf
```

### 7.4 firewalld (High-Level Firewall)

```bash
# firewalld uses zones for network trust levels

# Common zones:
  drop       - Drop all incoming, allow outgoing
  block      - Reject incoming, allow outgoing
  public     - Default zone, limited incoming
  external   - For routers with NAT
  internal   - Internal network, more trust
  dmz        - Limited access to internal
  work       - Work network
  home       - Home network
  trusted    - Accept all

# Commands
firewall-cmd --state                        # Check status
firewall-cmd --get-active-zones             # Active zones
firewall-cmd --list-all                     # List current zone rules
firewall-cmd --list-services                # List allowed services

# Add service
firewall-cmd --add-service=http             # Temporary
firewall-cmd --add-service=http --permanent # Permanent
firewall-cmd --reload                       # Apply permanent rules

# Add port
firewall-cmd --add-port=8080/tcp --permanent
firewall-cmd --add-port=5000-5100/udp --permanent

# Rich rules
firewall-cmd --add-rich-rule='rule family="ipv4" source address="192.168.1.0/24" service name="ssh" accept' --permanent
```

### 7.5 SELinux (Security-Enhanced Linux)

```bash
# SELinux modes:
  enforcing  - Enforce policies, deny violations
  permissive - Log violations but don't enforce
  disabled   - SELinux is off

# Check status
getenforce                        # Current mode
sestatus                          # Detailed status

# Change mode (temporary)
setenforce 0                      # Permissive
setenforce 1                      # Enforcing

# Change mode (permanent)
# Edit /etc/selinux/config
SELINUX=enforcing

# SELinux contexts
ls -Z /path/to/file               # View file context
ps auxZ                           # View process context

# Context format: user:role:type:level
# Example: system_u:object_r:httpd_sys_content_t:s0

# Change context
chcon -t httpd_sys_content_t /var/www/html/file
restorecon -R /var/www/html       # Restore default context

# SELinux booleans
getsebool -a                      # List all booleans
setsebool httpd_can_network_connect on  # Temporary
setsebool -P httpd_can_network_connect on  # Permanent

# Troubleshooting
ausearch -m avc -ts recent        # Search audit log
sealert -a /var/log/audit/audit.log  # Analyze alerts
audit2allow -M mypolicy < /var/log/audit/audit.log  # Generate policy
```

### 7.6 AppArmor (Alternative to SELinux)

```bash
# AppArmor uses profiles to restrict applications

# Check status
aa-status                         # AppArmor status
apparmor_status                   # Alias

# Modes:
  enforce    - Enforce profile restrictions
  complain   - Log violations but don't enforce

# Profile locations
/etc/apparmor.d/                  # Profile directory

# Profile management
aa-enforce /etc/apparmor.d/usr.bin.firefox   # Enforce profile
aa-complain /etc/apparmor.d/usr.bin.firefox  # Complain mode
aa-disable /etc/apparmor.d/usr.bin.firefox   # Disable profile

# Generate new profile
aa-genprof /path/to/program       # Interactive profile creation
aa-autodep /path/to/program       # Create base profile

# Reload profiles
apparmor_parser -r /etc/apparmor.d/profile.name
systemctl reload apparmor
```

### 7.7 SSH Security

```bash
# SSH configuration: /etc/ssh/sshd_config

# Key security settings:
Port 2222                         # Change default port
PermitRootLogin no                # Disable root login
PasswordAuthentication no         # Disable password auth
PubkeyAuthentication yes          # Enable key auth
MaxAuthTries 3                    # Limit auth attempts
LoginGraceTime 60                 # Timeout for login
AllowUsers john jane              # Whitelist users
AllowGroups sshusers              # Whitelist groups
Protocol 2                        # Use SSH protocol 2

# Apply changes
systemctl restart sshd

# SSH key management
ssh-keygen -t ed25519 -a 100      # Generate key (recommended)
ssh-keygen -t rsa -b 4096         # RSA alternative
ssh-copy-id user@server           # Copy public key to server

# Key file permissions
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519       # Private key
chmod 644 ~/.ssh/id_ed25519.pub   # Public key
chmod 600 ~/.ssh/authorized_keys  # Authorized keys

# Two-factor authentication
# Install google-authenticator and pam module
apt install libpam-google-authenticator
google-authenticator              # Setup for user

# /etc/pam.d/sshd
auth required pam_google_authenticator.so

# /etc/ssh/sshd_config
ChallengeResponseAuthentication yes
AuthenticationMethods publickey,keyboard-interactive
```

### 7.8 File Integrity Monitoring

```bash
# AIDE (Advanced Intrusion Detection Environment)

# Initialize database
aide --init
mv /var/lib/aide/aide.db.new /var/lib/aide/aide.db

# Check for changes
aide --check

# Update database after legitimate changes
aide --update

# Configuration: /etc/aide/aide.conf

# Tripwire (alternative)
tripwire --init                   # Initialize
tripwire --check                  # Check files
tripwire --update                 # Update database
```

### 7.9 Audit System

```bash
# auditd - Linux Audit Daemon

# Configuration
/etc/audit/auditd.conf            # Daemon config
/etc/audit/rules.d/               # Audit rules

# View audit log
ausearch -ts today                # Today's events
ausearch -k mykey                 # By key
aureport                          # Summary report
aureport -au                      # Authentication report

# Add audit rules
auditctl -w /etc/passwd -p wa -k passwd_changes  # Watch file
auditctl -w /etc/shadow -p wa -k shadow_changes
auditctl -w /var/log/auth.log -p r -k auth_read
auditctl -a exit,always -F arch=b64 -S execve -k command_execution

# Permanent rules in /etc/audit/rules.d/audit.rules
-w /etc/passwd -p wa -k passwd_changes
-w /etc/shadow -p wa -k shadow_changes
```

### 7.10 Security Scanning

```bash
# Lynis - Security auditing tool
lynis audit system                # Full system audit
lynis show details WARNING-001    # Show specific warning

# ClamAV - Antivirus
clamscan -r /path                 # Scan recursively
freshclam                         # Update virus database

# Rootkit detection
rkhunter --check                  # Check for rootkits
chkrootkit                        # Alternative tool

# OpenSCAP - Compliance checking
oscap xccdf eval --profile standard --report report.html /usr/share/xml/scap/ssg/content/ssg-ubuntu-ds.xml
```

### 7.11 System Hardening Checklist

```bash
# 1. Update system
apt update && apt upgrade -y

# 2. Remove unnecessary packages
apt autoremove

# 3. Disable root login
passwd -l root
# Edit /etc/ssh/sshd_config: PermitRootLogin no

# 4. Configure sudo
# Ensure users are in sudo group, don't use root

# 5. Set up firewall
ufw enable
ufw default deny incoming
ufw allow ssh

# 6. Configure automatic updates
apt install unattended-upgrades
dpkg-reconfigure unattended-upgrades

# 7. Disable unnecessary services
systemctl list-unit-files --type=service --state=enabled
systemctl disable service_name

# 8. Secure shared memory
# Add to /etc/fstab:
tmpfs /run/shm tmpfs defaults,noexec,nosuid 0 0

# 9. Configure sysctl security
# /etc/sysctl.d/99-security.conf
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.all.accept_source_route = 0
net.ipv4.icmp_echo_ignore_broadcasts = 1
kernel.randomize_va_space = 2
fs.suid_dumpable = 0

# 10. Set file permissions
chmod 600 /etc/shadow
chmod 644 /etc/passwd
chmod 700 /root
find /home -maxdepth 1 -type d -exec chmod 700 {} \;

# 11. Configure logging
# Ensure rsyslog or systemd-journald is running
# Forward logs to central server if possible

# 12. Enable and configure audit
systemctl enable auditd
systemctl start auditd

# 13. Set up intrusion detection
apt install aide
aide --init
```

### 7.12 Namespaces and Containers

```bash
# Linux namespaces provide isolation:

Namespace    Isolates
──────────────────────────────────
Mount (mnt)  Filesystem mount points
UTS          Hostname and domain name
IPC          Inter-process communication
Network      Network stack
PID          Process IDs
User         User and group IDs
Cgroup       Cgroup root directory

# View namespaces
lsns                              # List all namespaces
ls -la /proc/PID/ns/              # Process namespaces

# Create namespace (unshare)
unshare --net bash                # New network namespace
unshare --pid --fork bash         # New PID namespace

# Enter namespace (nsenter)
nsenter -t PID -n ip addr         # Enter network namespace

# Cgroups v2 hierarchy
/sys/fs/cgroup/
├── cgroup.controllers            # Available controllers
├── cgroup.subtree_control        # Enabled controllers
├── cpu.max                       # CPU limit
├── memory.max                    # Memory limit
├── io.max                        # I/O limit
└── pids.max                      # Process limit

# Create cgroup
mkdir /sys/fs/cgroup/mygroup
echo "100000 100000" > /sys/fs/cgroup/mygroup/cpu.max  # 100% of 1 CPU
echo $$ > /sys/fs/cgroup/mygroup/cgroup.procs          # Add current process
```

---

## Quick Reference Commands

```bash
# System Information
uname -a                  # Kernel info
lsb_release -a            # Distribution info
hostnamectl               # Hostname and OS info
uptime                    # System uptime

# Hardware
lscpu                     # CPU info
lsmem                     # Memory info
lspci                     # PCI devices
lsusb                     # USB devices
lsblk                     # Block devices

# Process Management
ps aux                    # All processes
top / htop                # Interactive process viewer
kill PID                  # Kill process
killall name              # Kill by name

# Disk and Filesystem
df -h                     # Disk usage
du -sh path               # Directory size
mount                     # Mounted filesystems
fdisk -l                  # Partition table

# Network
ip addr                   # IP addresses
ip route                  # Routing table
ss -tulpn                 # Open ports
netstat -tulpn            # Open ports (legacy)

# User Management
useradd / userdel         # Add/remove users
passwd user               # Change password
usermod -aG group user    # Add to group

# Service Management
systemctl status service  # Check status
systemctl start/stop      # Start/stop service
systemctl enable/disable  # Enable/disable at boot
journalctl -u service     # View logs

# Security
iptables -L               # Firewall rules
ufw status                # UFW status
getenforce                # SELinux mode
aa-status                 # AppArmor status
```

---

## Further Learning Resources

### Books
1. "Linux Kernel Development" by Robert Love
2. "Understanding the Linux Kernel" by Bovet & Cesati
3. "The Linux Command Line" by William Shotts
4. "Linux System Programming" by Robert Love
5. "Linux Security Cookbook" by Daniel Barrett

### Online Resources
1. Linux Documentation Project (tldp.org)
2. Kernel.org documentation
3. Red Hat System Administration guides
4. ArchWiki (wiki.archlinux.org)
5. Linux Journey (linuxjourney.com)

### Practice
1. Set up virtual machines with different distributions
2. Build Linux From Scratch (LFS)
3. Compile your own kernel
4. Create systemd services
5. Set up a firewall from scratch
6. Implement user management policies

---

*This guide provides a comprehensive overview of Linux internals. Each topic can be explored in much greater depth through hands-on practice and further reading.*
