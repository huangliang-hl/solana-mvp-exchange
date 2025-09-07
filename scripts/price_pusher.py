#!/usr/bin/env python3
"""
简单的价格推送脚本，用于演示订单触发
"""

import asyncio
import websockets
import json
import time
from typing import Dict, Any

class PricePusher:
    def __init__(self, url: str):
        self.url = url
        self.sequence_id = 1
        
    async def connect_and_push(self):
        """连接到WebSocket并推送价格数据"""
        try:
            async with websockets.connect(self.url) as websocket:
                print(f"✅ 已连接到价格服务: {self.url}")
                
                # 推送一系列价格来演示触发
                await self.push_price_sequence(websocket)
                
        except Exception as e:
            print(f"❌ 连接失败: {e}")
            
    async def push_price_sequence(self, websocket):
        """推送价格序列以触发订单"""
        
        # 价格序列：精确触发我们创建的订单
        # 买单：0.5 SOL，触发价格 ≥28.0，滑点容忍3.0%
        # 卖单：1.0 SOL，触发价格 ≤32.0，滑点容忍3.0%
        price_sequence = [
            {"price": 30.0, "description": "初始价格（无触发）", "delay": 1},
            {"price": 29.5, "description": "轻微下跌", "delay": 1},
            {"price": 28.5, "description": "接近买单触发价28.0", "delay": 1},
            {"price": 28.0, "description": "🎯 触发买单条件（≥28.0）", "delay": 0.5},
            {"price": 27.5, "description": "继续下跌，确认买单触发", "delay": 0.5},
            {"price": 27.0, "description": "最低点", "delay": 0.5},
            {"price": 28.0, "description": "开始反弹", "delay": 0.5},
            {"price": 29.5, "description": "回升", "delay": 0.5},
            {"price": 31.0, "description": "继续上涨", "delay": 0.5},
            {"price": 31.8, "description": "接近卖单触发价32.0", "delay": 0.5},
            {"price": 32.0, "description": "🎯 触发卖单条件（≤32.0）", "delay": 0.5},
            {"price": 33.0, "description": "继续上涨，确认卖单触发", "delay": 0.5},
            {"price": 34.0, "description": "最高点", "delay": 0.5},
        ]
        
        print(f"📈 开始推送价格序列，共 {len(price_sequence)} 个价格点")
        print(f"🎯 目标触发条件：")
        print(f"   📊 买单：价格 ≥28.0 USDC，数量 0.5 SOL，滑点容忍 3.0%")
        print(f"   📊 卖单：价格 ≤32.0 USDC，数量 1.0 SOL，滑点容忍 3.0%")
        print("")
        
        for i, price_data in enumerate(price_sequence):
            price = price_data["price"]
            description = price_data["description"]
            delay = price_data.get("delay", 2)
            
            await self.push_price(websocket, price)
            
            # 特殊标记触发点
            if "🎯" in description:
                print(f"🎯 [{i+1:2d}/{len(price_sequence)}] 价格: {price:6.2f} USDC - {description}")
            else:
                print(f"📊 [{i+1:2d}/{len(price_sequence)}] 价格: {price:6.2f} USDC - {description}")
            
            # 触发点延迟更长，让系统有时间处理
            await asyncio.sleep(delay)
            
    async def push_price(self, websocket, mid_price: float):
        """推送单个价格数据"""
        spread = 0.1  # 价差
        bid_price = mid_price - spread/2
        ask_price = mid_price + spread/2
        
        message = {
            "sequence_id": self.sequence_id,
            "timestamp": int(time.time() * 1000),
            "symbol": "SOL/USDC",
            "bid_price": f"{bid_price:.2f}",
            "ask_price": f"{ask_price:.2f}",
            "mid_price": f"{mid_price:.2f}",
            "source": "demo_script"
        }
        
        try:
            await websocket.send(json.dumps(message))
            self.sequence_id += 1
        except Exception as e:
            print(f"❌ 发送价格失败: {e}")

async def main():
    """主函数"""
    print("🚀 启动价格推送演示")
    print("💡 这将推送一系列价格来触发创建的订单")
    print("")
    
    # WebSocket URL
    ws_url = "ws://localhost:8081/price"
    
    pusher = PricePusher(ws_url)
    await pusher.connect_and_push()
    
    print("")
    print("✅ 价格推送演示完成")
    print("📊 请检查订单状态以确认触发和执行")

if __name__ == "__main__":
    asyncio.run(main())
