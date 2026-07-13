import { Canvas, useFrame } from '@react-three/fiber'
import { Environment, Html, Line, OrbitControls, Sparkles } from '@react-three/drei'
import { useMemo, useRef } from 'react'
import type { Group, Mesh } from 'three'
import { AdditiveBlending, Color } from 'three'
import { statusColor } from './demoCluster'
import type { ClusterService, HealthStatus } from './types'

interface ClusterSceneProps {
  services: ClusterService[]
  selectedId: string
  onSelect: (serviceId: string) => void
}

const connectionStatus = (source: ClusterService, target: ClusterService): HealthStatus => {
  if (source.status === 'unhealthy' || target.status === 'unhealthy') return 'unhealthy'
  if (source.status === 'degraded' || target.status === 'degraded') return 'degraded'
  if (source.status === 'starting' || target.status === 'starting') return 'starting'
  return 'healthy'
}

function Signal({
  start,
  end,
  status,
  offset,
}: {
  start: [number, number, number]
  end: [number, number, number]
  status: HealthStatus
  offset: number
}) {
  const ref = useRef<Mesh>(null)

  useFrame(({ clock }) => {
    if (!ref.current) return
    const progress = (clock.elapsedTime * 0.22 + offset) % 1
    ref.current.position.set(
      start[0] + (end[0] - start[0]) * progress,
      start[1] + (end[1] - start[1]) * progress + Math.sin(progress * Math.PI) * 0.22,
      start[2] + (end[2] - start[2]) * progress,
    )
  })

  return (
    <mesh ref={ref}>
      <sphereGeometry args={[0.035, 10, 10]} />
      <meshBasicMaterial color={statusColor[status]} blending={AdditiveBlending} transparent opacity={0.9} />
    </mesh>
  )
}

function Connection({ source, target }: { source: ClusterService; target: ClusterService }) {
  const status = connectionStatus(source, target)
  const midpoint: [number, number, number] = [
    (source.position[0] + target.position[0]) / 2,
    (source.position[1] + target.position[1]) / 2 + 0.22,
    (source.position[2] + target.position[2]) / 2,
  ]
  const points = [source.position, midpoint, target.position]

  return (
    <group>
      <Line
        points={points}
        color={statusColor[status]}
        lineWidth={status === 'healthy' ? 0.45 : 1.15}
        transparent
        opacity={status === 'healthy' ? 0.2 : 0.65}
      />
      <Signal start={source.position} end={target.position} status={status} offset={source.position[0] * 0.13 + 0.6} />
    </group>
  )
}

function ServiceNode({
  service,
  selected,
  onSelect,
}: {
  service: ClusterService
  selected: boolean
  onSelect: () => void
}) {
  const group = useRef<Group>(null)
  const core = useRef<Mesh>(null)
  const color = statusColor[service.status]

  useFrame(({ clock }, delta) => {
    if (group.current) group.current.rotation.y += delta * (selected ? 0.34 : 0.13)
    if (core.current) {
      const pulse = 1 + Math.sin(clock.elapsedTime * 2.2 + service.position[0]) * 0.035
      core.current.scale.setScalar(pulse)
    }
  })

  return (
    <group position={service.position}>
      <group
        ref={group}
        onClick={event => {
          event.stopPropagation()
          onSelect()
        }}
        onPointerOver={() => { document.body.style.cursor = 'pointer' }}
        onPointerOut={() => { document.body.style.cursor = 'default' }}
      >
        <mesh ref={core}>
          <icosahedronGeometry args={[selected ? 0.42 : 0.36, 4]} />
          <meshPhysicalMaterial
            color={new Color(color)}
            emissive={new Color(color)}
            emissiveIntensity={service.status === 'healthy' ? 0.25 : 1.15}
            roughness={0.16}
            metalness={0.15}
            transmission={0.35}
            transparent
            opacity={0.9}
          />
        </mesh>

        <mesh rotation={[Math.PI / 2, 0, 0]}>
          <torusGeometry args={[selected ? 0.69 : 0.58, 0.012, 8, 80]} />
          <meshBasicMaterial color={color} transparent opacity={selected ? 0.75 : 0.22} />
        </mesh>

        <mesh rotation={[0.4, 0.7, 0]}>
          <torusGeometry args={[selected ? 0.81 : 0.69, 0.006, 8, 80]} />
          <meshBasicMaterial color={color} transparent opacity={selected ? 0.38 : 0.12} />
        </mesh>

        {service.instances.map((instance, index) => {
          const angle = (index / service.instances.length) * Math.PI * 2 + 0.3
          const radius = 0.7 + (index % 2) * 0.12
          const instanceColor = statusColor[instance.status]
          return (
            <group key={instance.id} position={[Math.cos(angle) * radius, Math.sin(angle) * radius, Math.sin(angle * 1.8) * 0.22]}>
              <mesh>
                <sphereGeometry args={[0.11, 18, 18]} />
                <meshStandardMaterial
                  color={instanceColor}
                  emissive={instanceColor}
                  emissiveIntensity={instance.status === 'healthy' ? 0.4 : 1.6}
                />
              </mesh>
              <Line points={[[0, 0, 0], [-Math.cos(angle) * radius * 0.58, -Math.sin(angle) * radius * 0.58, 0]]} color={instanceColor} lineWidth={0.35} transparent opacity={0.28} />
            </group>
          )
        })}
      </group>

      <Html center position={[0, -1.05, 0]} distanceFactor={9} style={{ pointerEvents: 'none' }}>
        <div className={`node-label ${selected ? 'node-label--selected' : ''}`}>
          <span>{service.name}</span>
          <small>{service.instances.length} {service.instances.length === 1 ? 'instance' : 'instances'}</small>
        </div>
      </Html>
    </group>
  )
}

function SceneContent({ services, selectedId, onSelect }: ClusterSceneProps) {
  const connections = useMemo(() => services.flatMap(source => source.dependencies.flatMap(targetId => {
    const target = services.find(service => service.id === targetId)
    return target ? [{ source, target }] : []
  })), [services])

  return (
    <>
      <ambientLight intensity={0.45} />
      <pointLight position={[-5, 5, 6]} intensity={26} color="#c7dcff" distance={22} />
      <pointLight position={[5, -4, 3]} intensity={18} color="#ffffff" distance={18} />
      <Sparkles count={85} scale={[13, 8, 6]} size={1.25} speed={0.12} opacity={0.18} color="#a1a1aa" />
      {connections.map(({ source, target }) => (
        <Connection key={`${source.id}-${target.id}`} source={source} target={target} />
      ))}
      {services.map(service => (
        <ServiceNode
          key={service.id}
          service={service}
          selected={service.id === selectedId}
          onSelect={() => onSelect(service.id)}
        />
      ))}
      <Environment preset="city" environmentIntensity={0.2} />
      <OrbitControls
        enablePan={false}
        minDistance={7}
        maxDistance={17}
        autoRotate
        autoRotateSpeed={0.22}
        dampingFactor={0.055}
        minPolarAngle={Math.PI * 0.25}
        maxPolarAngle={Math.PI * 0.72}
      />
    </>
  )
}

export default function ClusterScene(props: ClusterSceneProps) {
  return (
    <Canvas
      camera={{ position: [0, 0.3, 11], fov: 46 }}
      dpr={[1, 1.75]}
      gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
      onPointerMissed={() => props.onSelect('api')}
    >
      <SceneContent {...props} />
    </Canvas>
  )
}
