use bpaf::Bpaf;
use std::cmp::Reverse;
use std::collections::BinaryHeap;
use std::io::{self, Write};
use std::net::{IpAddr, Ipv4Addr};
use std::sync::mpsc::{channel, Sender};
use std::sync::Arc;
use tokio::net::TcpStream;
use tokio::sync::Semaphore;
use tokio::task;

// Max IP Port.
const MAX: u16 = 65535;

// Address fallback.
const IPFALLBACK: IpAddr = IpAddr::V4(Ipv4Addr::new(127, 0, 0, 1));

// CLI Arguments.
#[derive(Debug, Clone, Bpaf)]
#[bpaf(options)]
pub struct Arguments {
    /// The address to scan.
    #[bpaf(long, short, argument("Address"), fallback(IPFALLBACK))]
    pub address: IpAddr,
    /// The first port to scan.
    #[bpaf(
        long("start"),
        short('s'),
        guard(start_port_guard, "Must be greater than 0"),
        fallback(1u16)
    )]
    pub start_port: u16,
    /// The last port to scan.
    #[bpaf(
        long("end"),
        short('e'),
        guard(end_port_guard, "Must be less than or equal to 65535"),
        fallback(MAX)
    )]
    pub end_port: u16,
}

fn start_port_guard(input: &u16) -> bool {
    *input > 0
}

fn end_port_guard(input: &u16) -> bool {
    *input <= MAX
}

// Scan a single port on a single IP address.
async fn scan(semaphore: Arc<Semaphore>, tx: Sender<u16>, start_port: u16, addr: IpAddr) {
    // Acquire a permit from the semaphore.
    let _permit = semaphore.acquire().await;

    // Connect to the port.
    match TcpStream::connect(format!("{}:{}", addr, start_port)).await {
        // If the port is open...
        Ok(_) => {
            // Print a dot to show progress.
            print!(".");
            io::stdout().flush().unwrap();

            // Send the open port number to the channel.
            tx.send(start_port).unwrap();
        }
        // Ignore errors.
        Err(_) => {}
    }
}

#[tokio::main]
async fn main() {
    let opts = arguments().run();
    let (tx, rx) = channel();
    let semaphore = Arc::new(Semaphore::new(1000)); // limit concurrency to 1000 tasks at a time
    for i in opts.start_port..opts.end_port {
        let tx = tx.clone();
        let semaphore = Arc::clone(&semaphore);
        task::spawn(async move { scan(semaphore, tx, i, opts.address).await });
    }
    drop(tx);
    let mut out = BinaryHeap::new();
    for p in rx {
        out.push(Reverse(p));
    }
    println!("");
    while let Some(Reverse(port)) = out.pop() {
        println!("{} is open", port);
    }
}
