use tokio::signal;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    println!("Cell Worker Started");
    
    // TODO: Implement gRPC server here
    
    // Keep alive until SIGTERM
    match signal::ctrl_c().await {
        Ok(()) => {},
        Err(err) => {
            eprintln!("Unable to listen for shutdown signal: {}", err);
        },
    }
    println!("Cell Worker Shutting Down");
    Ok(())
}
