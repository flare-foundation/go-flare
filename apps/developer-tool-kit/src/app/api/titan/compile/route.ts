import { compileSoliditySource } from "@/lib/titan/compile-contract";

export async function POST(request: Request) {
  try {
    const body = (await request.json()) as { source?: string; fileName?: string };
    const source = body.source?.trim();

    if (!source) {
      return Response.json({ ok: false, errors: ["Source code is required."] }, { status: 400 });
    }

    const result = compileSoliditySource(source, body.fileName ?? "Contract.sol");
    return Response.json(result);
  } catch (error) {
    return Response.json(
      {
        ok: false,
        errors: [error instanceof Error ? error.message : "Compilation request failed."],
      },
      { status: 500 },
    );
  }
}