import { useState } from "react";
import { call, useHfdCall } from "@hfd/react";

interface SysInfo {
	hostname: string;
	os: string;
	arch: string;
	cpus: number;
	goVersion: string;
}

export default function App() {
	const sys = useHfdCall<null, SysInfo>("SysInfo");
	const [name, setName] = useState("Hyggshi");
	const [greeted, setGreeted] = useState("");

	const greet = async () => {
		const res = (await call("Greet", { name })) as { message: string };
		setGreeted(res.message);
	};

	return (
		<main className="app">
			<h1>react-demo <span className="tag">HFD + React + Vite</span></h1>
			<p className="sub">useHfdCall calls SysInfo on mount — over native IPC.</p>

			<section className="card">
				{sys.loading && <p>loading SysInfo…</p>}
				{sys.error && <p className="err">SysInfo failed: {String(sys.error.message)}</p>}
				{sys.data && (
					<pre className="output">
	{sys.data.hostname} • {sys.data.os}/{sys.data.arch} • {sys.data.cpus} CPUs • {sys.data.goVersion}
					</pre>
				)}
			</section>

			<section className="card">
				<input value={name} onChange={(e) => setName(e.target.value)} />
				<button onClick={greet}>Greet</button>
				{greeted && <div className="output">{greeted}</div>}
			</section>
		</main>
	);
}
