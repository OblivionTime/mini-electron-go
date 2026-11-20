<script setup>
import { ref, onMounted } from 'vue'
import { invoke, on } from './bridge.js'
import { App } from './bindings/index.js'

const name = ref('MiniBlink')
const greeting = ref('')
const sum = ref(null)
const now = ref('')
const ticks = ref([])
const busy = ref(false)
const error = ref('')

async function run(fn) {
  error.value = ''
  try {
    await fn()
  } catch (e) {
    error.value = e.message
  }
}

function callGreet() {
  run(async () => {
    busy.value = true
    try {
      greeting.value = await App.Greet(name.value)
    } finally {
      busy.value = false
    }
  })
}

function callAdd() {
  run(async () => {
    sum.value = await invoke('App.Add', 2, 40)
  })
}

function callNow() {
  run(async () => {
    now.value = await invoke('App.Now')
  })
}

function openWindow() {
  run(() => invoke('App.OpenWindow'))
}

onMounted(() => {
  on('tick', (data) => {
    ticks.value.unshift(data)
    if (ticks.value.length > 8) ticks.value.pop()
  })
  run(async () => {
    if (!await App.StartTicker()) return
    if (await App.Greet('smoke') !== 'Hello, smoke from Go!') throw new Error('unexpected greeting')
    let rejected = false
    try {
      await App.Greet(42)
    } catch (err) {
      rejected = err.message.startsWith('arg 0:')
    }
    if (!rejected) throw new Error('invalid argument was not rejected')
    await App.SmokePassed()
  })
})
</script>

<template>
  <main>
    <h1>MiniBlink + Vue 3</h1>
    <p class="sub">&lt;script setup&gt; · mbQuery RPC · Go 事件推送</p>

    <p class="err" v-if="error">Error: {{ error }}</p>

    <section>
      <h2>JS → Go：请求 / 响应</h2>
      <div class="row">
        <input v-model="name" placeholder="your name" />
        <button :disabled="busy" @click="callGreet">App.Greet(name)</button>
        <span class="out">{{ greeting }}</span>
      </div>
      <div class="row">
        <button @click="callAdd">App.Add(2, 40)</button>
        <span class="out">= {{ sum ?? '?' }}</span>
        <button @click="callNow">App.Now()</button>
        <span class="out">{{ now }}</span>
      </div>
      <div class="row">
        <button @click="openWindow">App.OpenWindow()</button>
        <span class="out">打开第二个窗口（多窗口）</span>
      </div>
    </section>

    <section>
      <h2>Go → JS：事件推送</h2>
      <ul class="ticks">
        <li v-for="t in ticks" :key="t.count">#{{ t.count }} · {{ t.time }}</li>
      </ul>
      <p class="out" v-if="!ticks.length">等待 Go 推送…</p>
    </section>
  </main>
</template>

<style>
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: "Segoe UI", system-ui, sans-serif;
  background: #0f1420;
  color: #e6ecf5;
}
main { max-width: 760px; margin: 0 auto; padding: 28px 20px 60px; }
h1 { margin: 0; font-size: 24px; }
h1::after {
  content: " · Wails-like on MiniBlink";
  font-size: 13px;
  font-weight: 400;
  color: #6f7c93;
}
.sub { color: #6f7c93; font-size: 13px; margin: 4px 0 22px; }
h2 { font-size: 15px; color: #8ea2c0; margin: 26px 0 10px; }
section { border-top: 1px solid #232c3d; padding-top: 6px; }
.row { display: flex; align-items: center; gap: 10px; margin: 8px 0; flex-wrap: wrap; }
input {
  background: #161d2c; border: 1px solid #2b3550; color: #e6ecf5;
  padding: 8px 10px; border-radius: 8px; outline: none; min-width: 180px;
}
input:focus { border-color: #3f6ef0; }
button {
  background: #3f6ef0; color: #fff; border: 0; padding: 8px 14px;
  border-radius: 8px; cursor: pointer; font-size: 13px;
}
button:hover { background: #517cf5; }
button:disabled { opacity: .5; cursor: default; }
.out { color: #7ee0a8; font-family: Consolas, monospace; font-size: 13px; }
.err { background: #3a1620; border: 1px solid #7a2740; color: #ff9bb0; padding: 8px 12px; border-radius: 8px; }
.ticks { list-style: none; padding: 0; margin: 6px 0; font-family: Consolas, monospace; font-size: 13px; }
.ticks li { padding: 4px 8px; border-radius: 6px; }
.ticks li:nth-child(odd) { background: #161d2c; }
</style>
