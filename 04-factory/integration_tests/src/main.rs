use base64::Engine as _;
use near_gas::NearGas;
use near_workspaces::types::NearToken;
use serde_json::json;

const WASM_PATH: &str = "../main.wasm";
const GAS: NearGas = NearGas::from_tgas(300);

async fn deploy_factory(
    worker: &near_workspaces::Worker<near_workspaces::network::Sandbox>,
    wasm: &[u8],
) -> anyhow::Result<near_workspaces::Contract> {
    let contract = worker.dev_deploy(wasm).await?;
    let result = contract
        .call("init")
        .args_json(json!({}))
        .gas(GAS)
        .transact()
        .await?;
    println!("  init logs: {:?}", result.logs());
    assert!(result.is_success(), "factory init failed: {:?}", result);
    Ok(contract)
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let worker = near_workspaces::sandbox().await?;
    let wasm = std::fs::read(WASM_PATH)?;

    let block = worker.view_block().await?;
    let now_ms = block.timestamp() / 1_000_000;
    let future_end_ms = now_ms + 86_400_000;

    let user = worker.dev_create_account().await?;
    let auctioneer = worker.dev_create_account().await?;
    let ft_contract = worker.dev_create_account().await?;
    let nft_contract = worker.dev_create_account().await?;

    // ── Test 1: Init ──────────────────────────────────────────────
    println!("\n[1] Factory init");
    let factory = deploy_factory(&worker, &wasm).await?;

    let code_size = factory
        .view("get_code_size")
        .args_json(json!({}))
        .await?
        .json::<i64>()?;
    assert!(code_size > 0, "embedded auction.wasm should not be empty");
    println!("  OK code_size={code_size}");

    // ── Test 2: UpdateAuctionContract — unauthorized ───────────────
    println!("\n[2] UpdateAuctionContract — unauthorized caller");
    let fake_wasm: Vec<u8> = b"\x00asm\x01\x00\x00\x00".to_vec();
    let encoded = base64::engine::general_purpose::STANDARD.encode(&fake_wasm);

    let result = user
        .call(factory.id(), "update_auction_contract")
        .args_json(json!({ "code": encoded }))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Unauthorized update should be rejected");
    println!("  OK unauthorized update correctly rejected");

    // ── Test 3: UpdateAuctionContract — authorized ────────────────
    println!("\n[3] UpdateAuctionContract — authorized (contract calls itself)");
    let result = factory
        .as_account()
        .call(factory.id(), "update_auction_contract")
        .args_json(json!({ "code": encoded }))
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    assert!(result.is_success(), "Authorized update failed: {:?}", result);

    let new_size = factory
        .view("get_code_size")
        .args_json(json!({}))
        .await?
        .json::<i64>()?;
    assert_eq!(new_size, fake_wasm.len() as i64);
    println!("  OK code_size updated to {new_size}");

    // Restore original wasm for deploy tests
    let factory = deploy_factory(&worker, &wasm).await?;

    // ── Test 4: DeployNewAuction — insufficient deposit ───────────
    println!("\n[4] DeployNewAuction — insufficient deposit");
    let result = user
        .call(factory.id(), "deploy_new_auction")
        .args_json(json!({
            "name": "my-auction",
            "end_time": future_end_ms,
            "auctioneer": auctioneer.id(),
            "ft_contract": ft_contract.id(),
            "nft_contract": nft_contract.id(),
            "token_id": "token-1",
            "starting_price": "1000"
        }))
        .deposit(NearToken::from_yoctonear(1))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Insufficient deposit should be rejected");
    println!("  OK insufficient deposit correctly rejected");

    // ── Test 5: DeployNewAuction — success ────────────────────────
    println!("\n[5] DeployNewAuction — with sufficient deposit (2 NEAR)");
    let result = user
        .call(factory.id(), "deploy_new_auction")
        .args_json(json!({
            "name": "my-auction",
            "end_time": future_end_ms,
            "auctioneer": auctioneer.id(),
            "ft_contract": ft_contract.id(),
            "nft_contract": nft_contract.id(),
            "token_id": "token-1",
            "starting_price": "1000"
        }))
        .deposit(NearToken::from_near(2))
        .gas(GAS)
        .transact()
        .await?;
    println!("  logs: {:?}", result.logs());
    println!("  deploy is_success={}", result.is_success());
    assert!(result.is_success(), "DeployNewAuction failed: {:?}", result);
    println!("  OK auction subaccount deployed");

    // ── Test 6: DeployNewAuction — invalid name (too long) ────────
    println!("\n[6] DeployNewAuction — name too long");
    let long_name = "a".repeat(60);
    let result = user
        .call(factory.id(), "deploy_new_auction")
        .args_json(json!({
            "name": long_name,
            "end_time": future_end_ms,
            "auctioneer": auctioneer.id(),
            "ft_contract": ft_contract.id(),
            "nft_contract": nft_contract.id(),
            "token_id": "token-1",
            "starting_price": "1000"
        }))
        .deposit(NearToken::from_near(2))
        .gas(GAS)
        .transact()
        .await?;
    assert!(!result.is_success(), "Long name should be rejected");
    println!("  OK long name correctly rejected");

    println!("\n✓ All 04-factory integration tests passed");
    Ok(())
}
