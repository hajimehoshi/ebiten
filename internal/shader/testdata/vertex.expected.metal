struct Uniforms {
	float2 U0;
};

struct Attributes {
	float2 M0;
	float2 M1;
	float4 M2;
};

struct Varyings {
	float4 Position [[position]];
	float2 M0;
	float4 M1;
};

vertex Varyings Vertex(
	uint vid [[vertex_id]],
	const device Attributes* attributes [[buffer(0)]],
	constant Uniforms& uniforms [[buffer(1)]]) {
	Varyings varyings = {};
	float4x4 l0 = float4x4(0);
	l0 = float4x4((2.0) / ((uniforms.U0).x), 0.0, 0.0, 0.0, 0.0, (2.0) / ((uniforms.U0).y), 0.0, 0.0, 0.0, 0.0, 1.0, 0.0, -1.0, -1.0, 0.0, 1.0);
	varyings.Position = (l0) * (float4(attributes[vid].M0, 0.0, 1.0));
	varyings.M0 = attributes[vid].M1;
	varyings.M1 = attributes[vid].M2;
	return varyings;
}
