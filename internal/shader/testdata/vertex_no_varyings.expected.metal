struct Attributes {
	float2 M0;
	float2 M1;
	float4 M2;
};

struct Varyings {
	float4 Position [[position]];
};

vertex Varyings Vertex(
	uint vid [[vertex_id]],
	const device Attributes* attributes [[buffer(0)]]) {
	Varyings varyings = {};
	varyings.Position = float4(attributes[vid].M0, 0.0, 1.0);
	return varyings;
}

fragment float4 Fragment(
	Varyings varyings [[stage_in]],
	bool front_facing [[front_facing]]) {
	return float4(1.0);
}
