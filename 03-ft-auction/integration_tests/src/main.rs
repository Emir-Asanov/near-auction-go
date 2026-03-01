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
    ft_contract: &str,
    nft_contract: &str,
    token_id: &str,
    starting_price: &str,
) -> anyhow::Result<near_workspaces::Contract> {
    let contract = worker.dev_deploy(wasm).await?;
    let result = contract
        .call("init")
        .args_json(json!({
            "end_time": end_time_ms,
            "auctioneer": auctioneer,
            "ft_contract": ft_contract,
            "nft_contract": nft_contract,
            "token_id": token_id,
            "starting_price": starting_price
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
    let future_end_ms = now_ms + 86_400_000;

    let alice = worker.dev_create_account().await?;
    let bob = worker.dev_create_account().await?;
    let auctioneer = worker.dev_create_account().await?;
    let ft_account = worker.dev_create_account().await?;
    let nft_account = worker.dev_create_account().await?;

    // ── Test 1: Init ──────────────────────────────────────────────
    println!("\n[1] Init with ft_contract, nft_contract, starting_price");
    let contract = deploy_and_init(
        &worker,
        &wasm,
        future_end_ms,
        auctioneer.id().as_str(),
        ft_account.id().as_str(),
        nft_account.id().as_str(),
        "token-1",
        "1000",
    )
    .await?;

    let info = contract
        .view("get_auction_info")
        .args_json(json!({}))
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(info["ft_contract"].as_str().unwrap(), ft_account.id().as_str());
    assert_eq!(info["nft_contract"].as_str().unwrap(), nft_account.id().as_str());
    assert_eq!(info["token_id"].as_str().unwrap(), "token-1");
    assert_eq!(info["highest_bid"]["amount"].as_str().unwrap(), "1000");
    println!("  OK ft/nft contracts and starting_price stored");

    // ── Test 2: FT bid — Alice bids 2000 tokens ──────────────────
    println!("\n[2] FT bid — Alice sends 2000 tokens (called by ft_account)");
    let result = ft_account
        .call(contract.id(), "ft_on_transfer")
        .args_json(json!({
            "sender_id": alice.id(),
            "amount": "2000",
            "msg": ""
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    println!("  ft_on_transfer is_success={}", result.is_success());

    let bid = contract
        .view("get_highest_bid")
        .args_json(json!({}))
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(bid["bidder"].as_str().unwrap(), alice.id().as_str());
    assert_eq!(bid["amount"].as_str().unwrap(), "2000");
    println!("  OK Alice is highest bidder at 2000 tokens");

    // ── Test 3: Higher FT bid — Bob bids 3000 tokens ──────────────
    println!("\n[3] Higher FT bid — Bob sends 3000 tokens");
    let result = ft_account
        .call(contract.id(), "ft_on_transfer")
        .args_json(json!({
            "sender_id": bob.id(),
            "amount": "3000",
            "msg": ""
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    println!("  is_success={}", result.is_success());

    let bid = contract
        .view("get_highest_bid")
        .args_json(json!({}))
        .await?
        .json::<serde_json::Value>()?;
    assert_eq!(bid["bidder"].as_str().unwrap(), bob.id().as_str());
    assert_eq!(bid["amount"].as_str().unwrap(), "3000");
    println!("  OK Bob outbid Alice");

    // ── Test 4: Lower FT bid rejected ────────────────────────────
    println!("\n[4] Lower FT bid rejected — Alice sends 500 tokens");
    let result = ft_account
        .call(contract.id(), "ft_on_transfer")
        .args_json(json!({
            "sender_id": alice.id(),
            "amount": "500",
            "msg": ""
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Low FT bid should fail");
    println!("  OK low FT bid correctly rejected");

    // ── Test 5: Wrong FT contract rejected ───────────────────────
    println!("\n[5] Wrong FT contract rejected — called by alice (not ft_account)");
    let result = alice
        .call(contract.id(), "ft_on_transfer")
        .args_json(json!({
            "sender_id": alice.id(),
            "amount": "9999",
            "msg": ""
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Wrong FT contract should be rejected");
    println!("  OK wrong ft_contract correctly rejected");

    // ── Test 6: Claim before end rejected ─────────────────────────
    println!("\n[6] Claim before auction end");
    let result = alice
        .call(contract.id(), "claim")
        .args_json(json!({}))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Early claim should fail");
    println!("  OK early claim correctly rejected");

    // ── Test 7: FT bid on ended auction rejected ──────────────────
    println!("\n[7] FT bid after auction end — end_time=1");
    let ended = deploy_and_init(
        &worker,
        &wasm,
        1,
        auctioneer.id().as_str(),
        ft_account.id().as_str(),
        nft_account.id().as_str(),
        "token-1",
        "1000",
    )
    .await?;

    let result = ft_account
        .call(ended.id(), "ft_on_transfer")
        .args_json(json!({
            "sender_id": alice.id(),
            "amount": "5000",
            "msg": ""
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Bid on ended auction should fail");
    println!("  OK bid after end correctly rejected");

    // ── Test 8: Claim after end ───────────────────────────────────
    println!("\n[8] Claim after auction end");
    let result = alice
        .call(ended.id(), "claim")
        .args_json(json!({}))
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    println!("  claim is_success={}", result.is_success());

    let claimed = ended
        .view("get_claimed")
        .args_json(json!({}))
        .await?
        .json::<bool>()?;
    assert!(claimed, "get_claimed should be true after claim");
    println!("  OK claimed=true");

    // ── Test 9: Double claim rejected ────────────────────────────
    println!("\n[9] Double claim rejected");
    let result = alice
        .call(ended.id(), "claim")
        .args_json(json!({}))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Double claim should fail");
    println!("  OK double claim correctly rejected");

    println!("\n✓ All 03-ft-auction integration tests passed");
    Ok(())
}
