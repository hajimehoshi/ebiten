# Choose distance and integration units before tuning glow

Use this reference for custom 3D or volume shaders, not ordinary sprite effects.
Declare world units, camera axes, and the field's meaning. With Y-up, a horizontal
disk lies in XZ: radius is `length(p.xz)` and height is `p.y`. Z-up is equally
valid if camera, geometry, and lighting agree.

Do not conflate three different step rules:

- Sphere tracing relies on a conservative distance bound to a surface.
- Volume integration needs samples fine enough to resolve density changes.
- A curved ray needs a suitable integrator and curvature/error control; a
  straight-ray SDF bound alone does not make its steps safe.

Check reach as well as iteration count: 68 steps of length .08 cover only 5.44
world units. Intersect a bounding volume to skip empty space when applicable.
Report surface hits, domain exits, low transmittance, and iteration exhaustion
separately. An exhausted iteration budget is not proof that the ray escaped.
Compare against smaller steps and a larger budget at the most difficult view.

For an approximately constant source function over a volume segment:

```go
// Kage loop body. rho, sigmaT >= 0; ds > 0.
// Initialize radiance=vec3(0), transmittance=1 before the loop.
a := 1 - exp(-sigmaT*rho*ds)
radiance += transmittance*a*sourceColor
transmittance *= 1-a
```

Add `transmittance*background` at the end. For a transparent layer, radiance is
premultiplied and alpha is `1-transmittance`. This is an absorption/source-function
approximation, not a complete scattering or relativistic simulation. Independent
emission per unit distance needs its own integral, including the zero-extinction
limit; do not silently treat it as sourceColor.

Adding glow once per iteration without a step-length factor makes brightness
depend on sampling density. A signed negative distance can also turn
`exp(-distance*falloff)` into unbounded growth. Define whether the effect uses a
surface-distance band or a volume density. Establish normals and lighting with
glow disabled, then restore it under fixed exposure. A small coefficient from
one scene is not a universal safe value.

Camera proximity can hide the very outline being demonstrated. In ordinary
perspective, a sphere of radius R viewed from distance d>R has angular radius
`asin(R/d)`. Compare twice that angle with a full field of view. Orbiting at
constant distance while aiming at the center does not reduce a sphere's angular
size. Gravitational lensing has different apparent geometry.
