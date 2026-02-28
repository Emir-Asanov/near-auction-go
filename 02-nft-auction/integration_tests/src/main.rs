use near_gas::NearGas;
use near_workspaces::types::NearToken;
use serde_json::json;

const WASM_PATH: &str = "../main.wasm";
const GAS: NearGas = NearGas::from_tgas(300);

async fn deploy_and_init(
    worker: &near_workspaces::Worker<near_workspaces::network::Sandbox>,
    wasm: &[u8],
    end_time_ms: u64,
    auctioneer: &str,
    nft_contract: &str,
    token_id: &str,
) -> anyhow::Result<near_workspaces::Contract> {
    let contract = worker.dev_deploy(wasm).await?;
    let result = contract
        .call("init")
        .args_json(json!({
            "end_time": end_time_ms,
            "auctioneer": auctioneer,
            "nft_contract": nft_contract,
            "token_id": token_id
        }))
        .gas(GAS)
        .transact()
        .await?;
    println!("  init logs: {:?}", result.logs());
    assert!(result.is_success(), "init failed: {:?}", result);
    Ok(contract)
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let worker = near_workspaces::sandbox().await?;
    let wasm = std::fs::read(WASM_PATH)?;

    let block = worker.view_block().await?;
    let now_ms = block.timestamp() / 1_000_000;
    let future_end_ms = now_ms + 86_400_000; // 24 hours from now

    let alice = worker.dev_create_account().await?;
    let bob = worker.dev_create_account().await?;
    let auctioneer = worker.dev_create_account().await?;
    // Simulated NFT contract account (no real NFT logic needed for bid tests)
    let nft_contract = worker.dev_create_account().await?;

    // ── Test 1: Init ──────────────────────────────────────────────
    println!("\n[1] Init with nft_contract and token_id");
    let contract = deploy_and_init(
        &worker,
        &wasm,
        future_end_ms,
        auctioneer.id().as_str(),
        nft_contract.id().as_str(),
        "token-1",
    )
    .await?;

    let info = contract
        .view("get_auction_info")
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(info["auctioneer"].as_str().unwrap(), auctioneer.id().as_str());
    assert_eq!(
        info["nft_contract"].as_str().unwrap(),
        nft_contract.id().as_str()
    );
    assert_eq!(info["token_id"].as_str().unwrap(), "token-1");
    assert_eq!(info["claimed"].as_bool().unwrap(), false);
    println!("  OK nft_contract and token_id stored correctly");

    // ── Test 2: First bid ─────────────────────────────────────────
    println!("\n[2] First bid — Alice bids 2 NEAR");
    let result = alice
        .call(contract.id(), "bid")
        .deposit(NearToken::from_near(2))
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    assert!(result.is_success(), "Alice bid failed: {:?}", result);

    let bid = contract
        .view("get_highest_bid")
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(bid["bidder"].as_str().unwrap(), alice.id().as_str());
    println!("  OK Alice is highest bidder");

    // ── Test 3: Higher bid ────────────────────────────────────────
    println!("\n[3] Higher bid — Bob bids 3 NEAR");
    let result = bob
        .call(contract.id(), "bid")
        .deposit(NearToken::from_near(3))
        .gas(GAS)
        .transact()
        .await?;
    assert!(result.is_success(), "Bob bid failed: {:?}", result);

    let bid = contract
        .view("get_highest_bid")
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(bid["bidder"].as_str().unwrap(), bob.id().as_str());
    println!("  OK Bob outbid Alice");

    // ── Test 4: Lower bid rejected ────────────────────────────────
    println!("\n[4] Lower bid rejected — Alice bids 1 NEAR");
    let result = alice
        .call(contract.id(), "bid")
        .deposit(NearToken::from_near(1))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Low bid should have failed");
    println!("  OK low bid correctly rejected");

    // ── Test 5: Claim before end rejected ─────────────────────────
    println!("\n[5] Claim before auction end");
    let result = alice
        .call(contract.id(), "claim")
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Early claim should have failed");
    println!("  OK early claim correctly rejected");

    // ── Test 6: Bid after end rejected ───────────────────────────
    println!("\n[6] Bid after auction end — end_time=1");
    let ended = deploy_and_init(
        &worker,
        &wasm,
        1,
        auctioneer.id().as_str(),
        nft_contract.id().as_str(),
        "token-1",
    )
    .await?;

    let result = alice
        .call(ended.id(), "bid")
        .deposit(NearToken::from_near(5))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Bid on ended auction should fail");
    println!("  OK bid after end correctly rejected");

    // ── Test 7: Claim after end ───────────────────────────────────
    // Note: Claim sends NEAR to auctioneer and calls nft_transfer on the NFT
    // contract. The nft_transfer receipt will fail (no real NFT contract), but
    // the outer transaction (claim itself) should succeed and mark claimed=true.
    println!("\n[7] Claim after auction end");
    let result = alice
        .call(ended.id(), "claim")
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    // The outer tx succeeds; cross-contract nft_transfer may fail separately
    println!("  claim is_success={}", result.is_success());

    let claimed = ended.view("get_claimed").await?.json::<bool>()?;
    assert!(claimed, "get_claimed should be true after claim");
    println!("  OK claimed=true");

    // ── Test 8: Double claim rejected ────────────────────────────
    println!("\n[8] Double claim rejected");
    let result = alice
        .call(ended.id(), "claim")
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Double claim should fail");
    println!("  OK double claim correctly rejected");

    println!("\n✓ All 02-nft-auction integration tests passed");
    Ok(())
}
