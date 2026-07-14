import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:hankofield/features/my_seals/my_seals.dart';

void main() {
  setUpAll(sqfliteFfiInit);

  late Directory tempDirectory;
  late String databasePath;
  late LocalSealImageStore imageStore;
  late SqfliteLocalSealDesignRepository repository;

  setUp(() async {
    tempDirectory = await Directory.systemTemp.createTemp('hanko_seals_test_');
    databasePath = p.join(tempDirectory.path, 'local_seals.db');
    imageStore = LocalSealImageStore(
      documentsDirectoryProvider: () async => tempDirectory,
    );
    repository = SqfliteLocalSealDesignRepository(
      databaseFactory: databaseFactoryFfi,
      databasePathResolver: () async => databasePath,
      imageStore: imageStore,
    );
  });

  tearDown(() async {
    if (await tempDirectory.exists()) {
      await tempDirectory.delete(recursive: true);
    }
  });

  test('saves, lists, reads, and deletes local seal designs', () async {
    final firstImagePath = await imageStore.saveSealImage(
      localSealId: 'local seal:001',
      bytes: [0, 1, 2, 3],
    );
    final firstImage = File(firstImagePath);
    expect(await firstImage.exists(), isTrue);

    final firstDesign = _localSealDesign(localImagePath: firstImagePath);
    final secondDesign = _localSealDesign(
      id: 'local_seal_002',
      selectedKanji: '麗',
      aiVariantId: 'seal_variant_002',
      localImagePath: await imageStore.saveSealImage(
        localSealId: 'local_seal_002',
        bytes: [9, 8, 7],
      ),
      createdAt: DateTime.parse('2026-05-22T10:00:00+09:00'),
      updatedAt: DateTime.parse('2026-05-22T10:15:00+09:00'),
    );

    await repository.saveLocalSealDesign(firstDesign);
    await repository.saveLocalSealDesign(secondDesign);

    final listed = await repository.listLocalSealDesigns();
    expect(listed.map((design) => design.id), [
      'local_seal_002',
      'local_seal_001',
    ]);

    final saved = await repository.getLocalSealDesign('local_seal_001');
    expect(saved, isNotNull);
    expect(saved?.inputName, 'Michael');
    expect(saved?.selectedKanji, '美空');
    expect(saved?.meaning, 'Beautiful sky');
    expect(saved?.impression, ['Elegant', 'Gentle', 'Poetic']);
    expect(saved?.characterCount, 2);
    expect(saved?.strokeComplexity, 'medium');
    expect(saved?.engravingSuitability, 'high');
    expect(saved?.shape, 'square');
    expect(saved?.style, 'elegant');
    expect(saved?.strokeWeight, 'standard');
    expect(saved?.balance, 'balanced');
    expect(saved?.aiGenerationId, 'seal_request_001');
    expect(saved?.aiVariantId, 'seal_variant_001');
    expect(
      saved?.previewImageStoragePath,
      'seal_designs/seal_request_001/seal_variant_001.png',
    );
    expect(saved?.localImagePath, firstImagePath);
    expect(saved?.isFavorite, isFalse);

    await repository.deleteLocalSealDesign('local_seal_001');

    expect(await repository.getLocalSealDesign('local_seal_001'), isNull);
    expect(await firstImage.exists(), isFalse);
    expect(await repository.listLocalSealDesigns(), hasLength(1));
  });

  test(
    'stores seal image files safely within the app document directory',
    () async {
      final imagePath = await imageStore.saveSealImage(
        localSealId: '../../local seal:001',
        bytes: [4, 5, 6],
        fileExtension: '.PNG',
      );
      final imageDirectory = p.join(tempDirectory.path, 'seal_designs');
      final image = File(imagePath);

      expect(p.isWithin(imageDirectory, imagePath), isTrue);
      expect(p.basename(imagePath), isNot(contains('..')));
      expect(p.extension(imagePath), '.png');
      expect(await image.exists(), isTrue);
      expect(await image.readAsBytes(), [4, 5, 6]);

      final outsideImage = File(p.join(tempDirectory.path, 'outside.png'));
      await outsideImage.writeAsBytes([9, 9, 9], flush: true);

      await imageStore.deleteSealImage(outsideImage.path);
      expect(await outsideImage.exists(), isTrue);

      await imageStore.deleteSealImage(imagePath);
      expect(await image.exists(), isFalse);
    },
  );

  test('migrates legacy local seal schema before saving designs', () async {
    final legacyDb = await databaseFactoryFfi.openDatabase(
      databasePath,
      options: OpenDatabaseOptions(
        version: 1,
        onCreate: (db, _) async {
          await db.execute('''
CREATE TABLE local_seal_designs (
  id TEXT PRIMARY KEY,
  input_name TEXT NOT NULL,
  selected_kanji TEXT NOT NULL,
  reading TEXT NOT NULL,
  impression_json TEXT NOT NULL,
  character_count INTEGER NOT NULL,
  shape TEXT NOT NULL,
  style TEXT NOT NULL,
  stroke_weight TEXT NOT NULL,
  balance TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)
''');
          await db.insert('local_seal_designs', {
            'id': 'legacy_seal',
            'input_name': 'Eiko',
            'selected_kanji': '永',
            'reading': 'Ei',
            'impression_json': '["Timeless"]',
            'character_count': 0,
            'shape': 'square',
            'style': 'elegant',
            'stroke_weight': 'standard',
            'balance': 'balanced',
            'created_at': '2026-05-20T10:00:00.000+09:00',
            'updated_at': '2026-05-20T10:00:00.000+09:00',
          });
        },
      ),
    );
    await legacyDb.close();

    await repository.saveLocalSealDesign(
      _localSealDesign(id: 'local_seal_002', selectedKanji: '美嘉'),
    );

    final saved = await repository.getLocalSealDesign('local_seal_002');
    expect(saved?.selectedKanji, '美嘉');
    expect(saved?.previewImageStoragePath, isNotEmpty);

    final legacy = await repository.getLocalSealDesign('legacy_seal');
    expect(legacy?.selectedKanji, '永');
    expect(legacy?.characterCount, 1);
    expect(legacy?.aiGenerationId, 'legacy_seal');
    expect(legacy?.previewImageStoragePath, '');
  });

  test(
    'skips corrupted local seal rows instead of failing list reads',
    () async {
      await repository.saveLocalSealDesign(_localSealDesign());

      final db = await databaseFactoryFfi.openDatabase(databasePath);
      await db.insert('local_seal_designs', {
        'id': 'broken_seal',
        'input_name': 'Broken',
        'selected_kanji': '壊',
        'reading': 'Koware',
        'meaning': 'Broken',
        'impression_json': 'not json',
        'character_count': 1,
        'shape': 'square',
        'style': 'elegant',
        'stroke_weight': 'standard',
        'balance': 'balanced',
        'ai_generation_id': 'seal_request_broken',
        'ai_variant_id': 'seal_variant_broken',
        'preview_image_storage_path': 'seal_designs/broken.png',
        'preview_image_download_url': '',
        'local_image_path': p.join(tempDirectory.path, 'broken.png'),
        'is_favorite': 0,
        'created_at': 'not-a-date',
        'updated_at': 'not-a-date',
      });
      await db.close();

      final listed = await repository.listLocalSealDesigns();

      expect(listed.map((design) => design.id), ['local_seal_001']);
      expect(await repository.getLocalSealDesign('broken_seal'), isNull);
    },
  );
}

LocalSealDesign _localSealDesign({
  String id = 'local_seal_001',
  String selectedKanji = '美空',
  String aiVariantId = 'seal_variant_001',
  String localImagePath = '/tmp/local_seal_001.png',
  DateTime? createdAt,
  DateTime? updatedAt,
}) {
  return LocalSealDesign(
    id: id,
    inputName: 'Michael',
    selectedKanji: selectedKanji,
    reading: 'Misora',
    meaning: 'Beautiful sky',
    impression: const ['Elegant', 'Gentle', 'Poetic'],
    characterCount: selectedKanji.length,
    strokeComplexity: 'medium',
    engravingSuitability: 'high',
    shape: 'square',
    style: 'elegant',
    strokeWeight: 'standard',
    balance: 'balanced',
    aiGenerationId: 'seal_request_001',
    aiVariantId: aiVariantId,
    previewImageStoragePath: 'seal_designs/seal_request_001/$aiVariantId.png',
    previewImageDownloadUrl: 'https://storage.example.test/$aiVariantId.png',
    localImagePath: localImagePath,
    isFavorite: false,
    createdAt: createdAt ?? DateTime.parse('2026-05-21T11:00:00+09:00'),
    updatedAt: updatedAt ?? DateTime.parse('2026-05-21T11:10:00+09:00'),
  );
}
