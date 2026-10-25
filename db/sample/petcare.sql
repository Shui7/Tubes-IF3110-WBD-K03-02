/*M!999999\- enable the sandbox mode */ 
-- MariaDB dump 10.19-12.2.2-MariaDB, for Win64 (AMD64)
--
-- Host: localhost    Database: petcare
-- ------------------------------------------------------
-- Server version	12.2.2-MariaDB

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;

--
-- Table structure for table `adopsi`
--

DROP TABLE IF EXISTS `adopsi`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `adopsi` (
  `id_adopsi` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `id_hewan` int(11) NOT NULL,
  `waktu` datetime NOT NULL,
  `waktu_kembali` datetime DEFAULT NULL,
  PRIMARY KEY (`id_adopsi`),
  UNIQUE KEY `uq_adopsi` (`user_id`,`id_hewan`,`waktu`),
  KEY `fk_adopsi_hewan` (`id_hewan`),
  CONSTRAINT `fk_adopsi_adopter` FOREIGN KEY (`user_id`) REFERENCES `adopter` (`user_id`) ON UPDATE CASCADE,
  CONSTRAINT `fk_adopsi_hewan` FOREIGN KEY (`id_hewan`) REFERENCES `hewan` (`id_hewan`) ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=101 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adopsi`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `adopsi` WRITE;
/*!40000 ALTER TABLE `adopsi` DISABLE KEYS */;
INSERT INTO `adopsi` VALUES
(1,292,85,'2024-06-17 05:43:34','2025-06-12 05:43:34'),
(2,10,24,'2023-10-01 13:33:03',NULL),
(3,149,93,'2021-01-29 07:41:01',NULL),
(4,212,55,'2022-06-15 20:42:18',NULL),
(5,225,50,'2021-12-18 22:08:12',NULL),
(6,80,20,'2022-04-05 11:55:43',NULL),
(7,21,130,'2024-04-12 12:17:33',NULL),
(8,172,135,'2021-02-28 15:19:42',NULL),
(9,252,34,'2023-07-17 18:57:40',NULL),
(10,122,47,'2024-07-15 10:47:24',NULL),
(11,257,43,'2024-01-26 07:39:03',NULL),
(12,90,94,'2023-11-17 16:38:14',NULL),
(13,174,73,'2024-03-14 19:10:32',NULL),
(14,216,60,'2023-04-10 13:22:43',NULL),
(15,8,50,'2022-07-26 10:06:17',NULL),
(16,202,113,'2022-12-19 04:40:37',NULL),
(17,274,98,'2023-02-21 05:43:13',NULL),
(18,122,52,'2024-05-29 17:06:32',NULL),
(19,196,14,'2023-09-24 00:53:13',NULL),
(20,222,142,'2024-12-22 08:59:18','2025-09-22 08:59:18'),
(21,128,73,'2021-05-11 07:05:28',NULL),
(22,241,116,'2024-11-07 08:15:42',NULL),
(23,264,24,'2024-11-15 10:05:06',NULL),
(24,216,90,'2024-12-16 10:30:28','2025-01-22 10:30:28'),
(25,198,129,'2022-08-04 22:17:41',NULL),
(26,158,96,'2021-12-15 19:24:42','2022-02-13 19:24:42'),
(27,185,86,'2024-05-14 08:48:31','2024-06-12 08:48:31'),
(28,103,121,'2023-12-18 07:26:40',NULL),
(29,111,121,'2022-11-28 21:04:30',NULL),
(30,158,5,'2022-02-02 13:32:08',NULL),
(31,80,141,'2024-02-04 17:16:40',NULL),
(32,131,29,'2024-04-20 17:48:47',NULL),
(33,180,32,'2021-03-16 04:13:42',NULL),
(34,49,118,'2021-04-08 13:37:13','2021-12-29 13:37:13'),
(35,156,138,'2021-01-26 12:45:15',NULL),
(36,202,62,'2024-01-22 05:16:27','2024-09-07 05:16:27'),
(37,39,91,'2022-01-09 16:53:05',NULL),
(38,212,22,'2023-03-25 04:55:07',NULL),
(39,66,135,'2023-04-26 04:50:44',NULL),
(40,46,142,'2021-02-01 14:07:04',NULL),
(41,8,12,'2023-09-15 11:31:29',NULL),
(42,222,65,'2024-03-06 21:58:55','2024-04-19 21:58:55'),
(43,205,62,'2024-02-13 14:01:18',NULL),
(44,184,149,'2024-02-17 01:45:26',NULL),
(45,182,12,'2022-07-02 06:39:37',NULL),
(46,292,45,'2024-07-13 22:56:47',NULL),
(47,255,123,'2023-09-09 08:49:10','2023-11-27 08:49:10'),
(48,66,118,'2021-02-27 10:51:20',NULL),
(49,283,52,'2024-10-08 00:39:33','2025-03-25 00:39:33'),
(50,149,102,'2024-06-20 16:32:31',NULL),
(51,196,10,'2024-03-16 04:43:15',NULL),
(52,103,92,'2024-09-01 06:51:14',NULL),
(53,12,85,'2022-03-01 10:15:12','2022-09-15 10:15:12'),
(54,264,103,'2024-02-29 16:31:11',NULL),
(55,248,87,'2021-10-18 05:03:02',NULL),
(56,67,95,'2024-12-06 20:47:30',NULL),
(57,241,22,'2024-02-04 11:33:23',NULL),
(58,264,47,'2023-04-28 08:45:34',NULL),
(59,184,21,'2022-05-25 09:44:24',NULL),
(60,157,115,'2023-07-26 03:21:45',NULL),
(61,84,114,'2022-07-01 02:37:17',NULL),
(62,189,91,'2023-08-14 03:19:28',NULL),
(63,114,15,'2021-04-27 13:44:14',NULL),
(64,254,94,'2023-03-09 01:13:59',NULL),
(65,1,41,'2021-02-18 09:00:23','2022-01-05 09:00:23'),
(66,1,113,'2021-02-24 01:23:58','2021-07-04 01:23:58'),
(67,274,94,'2022-07-17 13:57:08',NULL),
(68,289,9,'2023-07-29 23:34:13','2024-03-25 23:34:13'),
(69,222,96,'2022-11-21 15:15:22',NULL),
(70,202,36,'2023-04-03 11:41:42',NULL),
(71,117,72,'2022-01-23 18:11:48',NULL),
(72,19,48,'2023-02-15 05:42:26',NULL),
(73,89,31,'2022-02-11 15:44:30',NULL),
(74,176,115,'2021-11-30 07:45:54',NULL),
(75,103,126,'2021-11-07 23:11:22','2021-12-24 23:11:22'),
(76,216,45,'2024-10-28 01:10:57',NULL),
(77,187,82,'2023-11-04 10:52:19',NULL),
(78,66,141,'2023-04-23 05:36:48',NULL),
(79,180,41,'2024-01-11 18:10:00',NULL),
(80,242,45,'2024-05-17 23:08:45',NULL),
(81,285,32,'2021-11-09 05:05:28',NULL),
(82,82,127,'2021-02-10 19:56:22',NULL),
(83,202,95,'2022-12-28 02:09:09',NULL),
(84,112,34,'2023-08-09 15:05:18',NULL),
(85,66,80,'2022-09-11 12:28:47',NULL),
(86,296,123,'2023-03-28 15:21:16',NULL),
(87,90,148,'2021-04-25 13:37:33','2021-10-14 13:37:33'),
(88,292,19,'2022-12-01 00:15:54',NULL),
(89,68,89,'2021-07-19 08:39:39',NULL),
(90,274,142,'2023-09-21 21:14:47',NULL),
(91,252,34,'2022-11-04 01:10:43',NULL),
(92,221,32,'2023-03-16 20:14:51','2023-06-18 20:14:51'),
(93,122,83,'2024-12-23 01:33:08',NULL),
(94,205,133,'2024-10-22 05:15:23',NULL),
(95,158,65,'2021-11-01 22:39:57',NULL),
(96,112,71,'2021-07-14 10:41:12',NULL),
(97,239,137,'2021-05-26 03:15:48',NULL),
(98,84,149,'2021-08-28 12:46:44','2022-07-23 12:46:44'),
(99,296,87,'2024-08-01 19:47:42',NULL),
(100,283,8,'2021-05-07 04:12:27','2022-04-10 04:12:27');
/*!40000 ALTER TABLE `adopsi` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `adopter`
--

DROP TABLE IF EXISTS `adopter`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `adopter` (
  `user_id` int(11) NOT NULL,
  `preferensi` varchar(200) DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  CONSTRAINT `fk_adopter_user` FOREIGN KEY (`user_id`) REFERENCES `pengguna` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `adopter`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `adopter` WRITE;
/*!40000 ALTER TABLE `adopter` DISABLE KEYS */;
INSERT INTO `adopter` VALUES
(1,'Hewan muda'),
(8,'Tidak ada preferensi'),
(10,'Hewan eksotis'),
(12,'Tidak ada preferensi'),
(13,'Hewan dewasa'),
(18,'Anjing besar'),
(19,'Hewan muda'),
(20,'Hewan muda'),
(21,'Semua jenis'),
(22,'Hewan muda'),
(28,'Kucing betina'),
(30,'Kucing betina'),
(39,'Hewan dewasa'),
(46,'Semua jenis'),
(49,'Semua jenis'),
(51,'Kucing indoor'),
(53,'Hewan muda'),
(56,'Hewan eksotis'),
(66,'Anjing besar'),
(67,'Tidak ada preferensi'),
(68,'Kucing betina'),
(72,'Kucing indoor'),
(76,'Anjing besar'),
(77,'Hewan dewasa'),
(80,'Anjing kecil'),
(81,'Anjing besar'),
(82,'Kucing betina'),
(84,'Hewan muda'),
(85,'Kucing betina'),
(89,'Hewan dewasa'),
(90,'Anjing besar'),
(91,'Semua jenis'),
(92,'Kucing betina'),
(94,'Hewan eksotis'),
(99,'Anjing kecil'),
(103,'Kucing indoor'),
(108,'Anjing besar'),
(111,'Hewan eksotis'),
(112,'Tidak ada preferensi'),
(114,'Hewan dewasa'),
(115,'Hewan eksotis'),
(117,'Semua jenis'),
(122,'Hewan muda'),
(128,'Hewan eksotis'),
(131,'Hewan eksotis'),
(134,'Kucing betina'),
(137,'Tidak ada preferensi'),
(138,'Semua jenis'),
(148,'Hewan muda'),
(149,'Kucing indoor'),
(153,'Tidak ada preferensi'),
(156,'Kucing betina'),
(157,'Semua jenis'),
(158,'Tidak ada preferensi'),
(165,'Semua jenis'),
(171,'Kucing betina'),
(172,'Tidak ada preferensi'),
(174,'Kucing indoor'),
(176,'Semua jenis'),
(180,'Hewan muda'),
(182,'Anjing besar'),
(184,'Anjing besar'),
(185,'Hewan muda'),
(187,'Hewan eksotis'),
(189,'Hewan dewasa'),
(190,'Tidak ada preferensi'),
(196,'Kucing indoor'),
(198,'Anjing besar'),
(202,'Tidak ada preferensi'),
(205,'Semua jenis'),
(207,'Kucing indoor'),
(209,'Tidak ada preferensi'),
(211,'Kucing betina'),
(212,'Anjing besar'),
(216,'Anjing besar'),
(221,'Hewan dewasa'),
(222,'Kucing betina'),
(225,'Anjing kecil'),
(230,'Tidak ada preferensi'),
(236,'Semua jenis'),
(238,'Semua jenis'),
(239,'Kucing betina'),
(241,'Tidak ada preferensi'),
(242,'Hewan muda'),
(246,'Hewan eksotis'),
(248,'Semua jenis'),
(252,'Semua jenis'),
(254,'Kucing betina'),
(255,'Hewan eksotis'),
(257,'Kucing betina'),
(264,'Anjing kecil'),
(274,'Anjing besar'),
(276,'Anjing kecil'),
(280,'Kucing indoor'),
(283,'Hewan muda'),
(285,'Semua jenis'),
(289,'Kucing betina'),
(292,'Hewan muda'),
(295,'Kucing betina'),
(296,'Hewan dewasa');
/*!40000 ALTER TABLE `adopter` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `anjing`
--

DROP TABLE IF EXISTS `anjing`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `anjing` (
  `id_hewan` int(11) NOT NULL,
  `ras` varchar(100) NOT NULL,
  PRIMARY KEY (`id_hewan`),
  CONSTRAINT `fk_anjing_hewan` FOREIGN KEY (`id_hewan`) REFERENCES `hewan` (`id_hewan`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `anjing`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `anjing` WRITE;
/*!40000 ALTER TABLE `anjing` DISABLE KEYS */;
INSERT INTO `anjing` VALUES
(4,'Dachshund'),
(14,'Chihuahua'),
(15,'Poodle'),
(16,'Bulldog'),
(18,'Beagle'),
(22,'Poodle'),
(24,'Labrador'),
(25,'Border Collie'),
(27,'Poodle'),
(29,'Maltese'),
(32,'Labrador'),
(34,'Samoyede'),
(36,'Beagle'),
(40,'Labrador'),
(45,'Rottweiler'),
(48,'Bulldog'),
(55,'Dachshund'),
(72,'Shih Tzu'),
(74,'Dachshund'),
(78,'Golden Retriever'),
(79,'Poodle'),
(80,'Rottweiler'),
(83,'Chihuahua'),
(85,'Bulldog'),
(87,'Border Collie'),
(89,'Shih Tzu'),
(91,'Dalmatian'),
(96,'Beagle'),
(98,'Bulldog'),
(99,'Samoyede'),
(104,'Border Collie'),
(105,'Labrador'),
(106,'Poodle'),
(109,'Poodle'),
(111,'Maltese'),
(112,'Labrador'),
(122,'Golden Retriever'),
(123,'Chihuahua'),
(124,'Golden Retriever'),
(126,'Rottweiler'),
(127,'Bulldog'),
(130,'Maltese'),
(131,'Chihuahua'),
(134,'Poodle'),
(135,'Maltese'),
(136,'Husky'),
(137,'Pomeranian'),
(138,'Golden Retriever'),
(141,'Beagle'),
(149,'Beagle');
/*!40000 ALTER TABLE `anjing` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `catatanmedis`
--

DROP TABLE IF EXISTS `catatanmedis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `catatanmedis` (
  `nomor_catatan` int(11) NOT NULL AUTO_INCREMENT,
  `waktu_buat` date NOT NULL,
  `ringkasan_kondisi` text DEFAULT NULL,
  `id_hewan` int(11) NOT NULL,
  PRIMARY KEY (`nomor_catatan`),
  UNIQUE KEY `id_hewan` (`id_hewan`),
  CONSTRAINT `fk_catmed_hewan` FOREIGN KEY (`id_hewan`) REFERENCES `hewan` (`id_hewan`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `catatanmedis`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `catatanmedis` WRITE;
/*!40000 ALTER TABLE `catatanmedis` DISABLE KEYS */;
INSERT INTO `catatanmedis` VALUES
(1,'2024-03-28','Sedang dalam pemulihan pasca operasi.',74),
(2,'2021-10-19','Sedang dalam pemulihan pasca operasi.',32),
(3,'2023-08-15','Hewan dalam kondisi sehat, tidak ada keluhan.',147),
(4,'2024-04-23','Ditemukan infeksi ringan pada telinga.',140),
(5,'2023-08-28','Terdeteksi parasit eksternal, sudah ditangani.',98),
(6,'2021-04-20','Hewan aktif dan responsif.',102),
(7,'2024-02-21','Hewan dalam kondisi sehat, tidak ada keluhan.',89),
(8,'2020-12-24','Kondisi gigi dan mulut baik.',38),
(9,'2024-12-11','Berat badan kurang, disarankan diet khusus.',75),
(10,'2020-03-15','Kondisi gigi dan mulut baik.',11),
(11,'2024-06-01','Perlu pemeriksaan lanjutan kulit.',150),
(12,'2021-01-16','Sedang dalam pemulihan pasca operasi.',21),
(13,'2024-11-30','Hewan dalam kondisi sehat, tidak ada keluhan.',144),
(14,'2020-01-19','Sedang dalam pemulihan pasca operasi.',114),
(15,'2023-03-09','Terdeteksi parasit eksternal, sudah ditangani.',66),
(16,'2020-08-07','Alergi terhadap beberapa jenis makanan.',123),
(17,'2021-08-05','Perlu pemeriksaan lanjutan kulit.',55),
(18,'2023-08-06','Hewan aktif dan responsif.',52),
(19,'2022-12-09','Perlu pemeriksaan lanjutan kulit.',70),
(20,'2024-12-26','Berat badan kurang, disarankan diet khusus.',132),
(21,'2024-10-08','Hewan aktif dan responsif.',36),
(22,'2022-08-16','Sedang dalam pemulihan pasca operasi.',28),
(23,'2021-01-29','Ditemukan infeksi ringan pada telinga.',62),
(24,'2021-11-08','Berat badan kurang, disarankan diet khusus.',149),
(25,'2024-02-06','Perlu pemeriksaan lanjutan kulit.',7),
(26,'2023-08-20','Sedang dalam pemulihan pasca operasi.',86),
(27,'2024-01-11','Berat badan kurang, disarankan diet khusus.',116),
(28,'2020-01-29','Alergi terhadap beberapa jenis makanan.',68),
(29,'2024-06-07','Sedang dalam pemulihan pasca operasi.',29),
(30,'2023-03-08','Riwayat vaksin lengkap.',82),
(31,'2020-12-25','Terdeteksi parasit eksternal, sudah ditangani.',30),
(32,'2024-10-19','Kondisi gigi dan mulut baik.',126),
(33,'2024-08-20','Kondisi gigi dan mulut baik.',13),
(34,'2022-11-21','Alergi terhadap beberapa jenis makanan.',53),
(35,'2020-06-26','Kondisi gigi dan mulut baik.',43),
(36,'2023-10-02','Ditemukan infeksi ringan pada telinga.',92),
(37,'2021-01-13','Berat badan kurang, disarankan diet khusus.',61),
(38,'2022-09-04','Alergi terhadap beberapa jenis makanan.',118),
(39,'2021-05-23','Hewan dalam kondisi sehat, tidak ada keluhan.',88),
(40,'2021-06-18','Kondisi gigi dan mulut baik.',99),
(41,'2021-04-27','Perlu pemeriksaan lanjutan kulit.',18),
(42,'2024-03-23','Sedang dalam pemulihan pasca operasi.',1),
(43,'2021-11-06','Sedang dalam pemulihan pasca operasi.',71),
(44,'2023-04-07','Terdeteksi parasit eksternal, sudah ditangani.',139),
(45,'2020-01-24','Sedang dalam pemulihan pasca operasi.',117),
(46,'2023-08-31','Alergi terhadap beberapa jenis makanan.',84),
(47,'2023-11-19','Riwayat vaksin lengkap.',137),
(48,'2020-05-07','Ditemukan infeksi ringan pada telinga.',128),
(49,'2022-08-11','Sedang dalam pemulihan pasca operasi.',105),
(50,'2020-11-23','Kondisi gigi dan mulut baik.',26);
/*!40000 ALTER TABLE `catatanmedis` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `dokterhewan`
--

DROP TABLE IF EXISTS `dokterhewan`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `dokterhewan` (
  `id_dokter` int(11) NOT NULL AUTO_INCREMENT,
  `nama` varchar(100) NOT NULL,
  `tempat_praktik` varchar(200) NOT NULL,
  PRIMARY KEY (`id_dokter`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `dokterhewan`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `dokterhewan` WRITE;
/*!40000 ALTER TABLE `dokterhewan` DISABLE KEYS */;
INSERT INTO `dokterhewan` VALUES
(1,'drh. Luluh Prayoga','Klinik Hewan Sihombing - Samarinda'),
(2,'drh. Siska Pangestu','Klinik Hewan Saputra - Bau-Bau'),
(3,'drh. Lala Waluyo','Klinik Hewan Marbun - Palembang'),
(4,'drh. Ophelia Suryatmi','Klinik Hewan Astuti - Pariaman'),
(5,'drh. Prabowo Wibowo','Klinik Hewan Aryani - Tual'),
(6,'drh. Rina Nasyiah','Klinik Hewan Mangunsong - Kota Administrasi Jakarta Pusat'),
(7,'drh. Malika Samosir','Klinik Hewan Hasanah - Metro'),
(8,'drh. Drs. Jaya Hidayanto','Klinik Hewan Sitorus - Tasikmalaya'),
(9,'drh. Faizah Mangunsong','Klinik Hewan Nainggolan - Balikpapan'),
(10,'drh. Drs. Titi Wahyudin, M.Kom.','Klinik Hewan Waskita - Mojokerto'),
(11,'drh. Kasiyah Winarno, M.Kom.','Klinik Hewan Samosir - Padang Sidempuan'),
(12,'drh. Viman Maheswara','Klinik Hewan Winarno - Depok'),
(13,'drh. dr. Yusuf Rajasa, S.I.Kom','Klinik Hewan Melani - Subulussalam'),
(14,'drh. Kartika Hakim, S.Sos','Klinik Hewan Anggraini - Kota Administrasi Jakarta Barat'),
(15,'drh. Ami Tarihoran','Klinik Hewan Usamah - Tanjungpinang'),
(16,'drh. Mariadi Maulana, S.T.','Klinik Hewan Suartini - Tangerang Selatan'),
(17,'drh. Dodo Prayoga','Klinik Hewan Suartini - Balikpapan'),
(18,'drh. Prasetya Iswahyudi','Klinik Hewan Yuniar - Mojokerto'),
(19,'drh. Rini Hutapea, M.TI.','Klinik Hewan Nainggolan - Cimahi'),
(20,'drh. drg. Hafshah Sihombing, M.TI.','Klinik Hewan Nasyidah - Semarang');
/*!40000 ALTER TABLE `dokterhewan` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `eksotis`
--

DROP TABLE IF EXISTS `eksotis`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `eksotis` (
  `id_hewan` int(11) NOT NULL,
  `jenis` varchar(100) NOT NULL,
  PRIMARY KEY (`id_hewan`),
  CONSTRAINT `fk_eksotis_hewan` FOREIGN KEY (`id_hewan`) REFERENCES `hewan` (`id_hewan`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `eksotis`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `eksotis` WRITE;
/*!40000 ALTER TABLE `eksotis` DISABLE KEYS */;
INSERT INTO `eksotis` VALUES
(2,'Tokek Leopard'),
(3,'Ular Piton'),
(5,'Ular Piton'),
(8,'Chinchilla'),
(10,'Landak Mini'),
(11,'Hamster'),
(12,'Landak Mini'),
(13,'Landak Mini'),
(17,'Tokek Leopard'),
(19,'Tokek Leopard'),
(28,'Tokek Leopard'),
(30,'Iguana'),
(31,'Sugar Glider'),
(33,'Tokek Leopard'),
(37,'Kura-kura'),
(44,'Sugar Glider'),
(46,'Kelinci'),
(47,'Kura-kura'),
(50,'Kelinci'),
(58,'Chinchilla'),
(61,'Landak Mini'),
(64,'Hamster'),
(66,'Kura-kura'),
(67,'Kura-kura'),
(70,'Ular Piton'),
(77,'Ular Piton'),
(81,'Landak Mini'),
(82,'Iguana'),
(84,'Sugar Glider'),
(86,'Burung Kakatua'),
(94,'Iguana'),
(97,'Kelinci'),
(100,'Chinchilla'),
(101,'Chinchilla'),
(103,'Landak Mini'),
(107,'Hamster'),
(108,'Sugar Glider'),
(116,'Iguana'),
(117,'Ular Piton'),
(118,'Iguana'),
(119,'Ular Piton'),
(120,'Tokek Leopard'),
(125,'Chinchilla'),
(128,'Hamster'),
(133,'Hamster'),
(140,'Ular Piton'),
(145,'Iguana'),
(146,'Kura-kura'),
(148,'Ular Piton'),
(150,'Hamster');
/*!40000 ALTER TABLE `eksotis` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `hewan`
--

DROP TABLE IF EXISTS `hewan`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `hewan` (
  `id_hewan` int(11) NOT NULL AUTO_INCREMENT,
  `nama` varchar(100) NOT NULL,
  `tgl_lahir` date NOT NULL,
  `jenis_kelamin` enum('Jantan','Betina') NOT NULL,
  `status` enum('Tersedia','Diadopsi','Karantina') NOT NULL DEFAULT 'Tersedia',
  `id_shelter` int(11) NOT NULL,
  PRIMARY KEY (`id_hewan`),
  KEY `fk_hewan_shelter` (`id_shelter`),
  CONSTRAINT `fk_hewan_shelter` FOREIGN KEY (`id_shelter`) REFERENCES `shelter` (`id_shelter`) ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=151 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `hewan`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `hewan` WRITE;
/*!40000 ALTER TABLE `hewan` DISABLE KEYS */;
INSERT INTO `hewan` VALUES
(1,'Mochi','2018-06-09','Betina','Tersedia',4),
(2,'Oreo','2023-02-02','Jantan','Diadopsi',4),
(3,'Koko','2023-05-01','Jantan','Tersedia',7),
(4,'Luna','2023-04-14','Betina','Diadopsi',19),
(5,'Putih','2023-07-11','Betina','Tersedia',7),
(6,'Hitam','2018-04-28','Jantan','Diadopsi',6),
(7,'Comel','2018-05-23','Jantan','Karantina',18),
(8,'Gendut','2016-06-02','Betina','Tersedia',2),
(9,'Manis','2021-03-16','Betina','Karantina',5),
(10,'Bontot','2022-02-25','Betina','Tersedia',1),
(11,'Cici','2021-06-09','Betina','Diadopsi',16),
(12,'Budi','2019-12-10','Betina','Tersedia',2),
(13,'Dora','2017-10-31','Betina','Tersedia',3),
(14,'Toto','2019-06-30','Betina','Tersedia',19),
(15,'Lili','2022-01-22','Jantan','Tersedia',5),
(16,'Panda','2021-04-24','Betina','Tersedia',8),
(17,'Nico','2016-04-30','Betina','Karantina',20),
(18,'Rara','2023-11-13','Jantan','Karantina',13),
(19,'Suri','2020-01-20','Betina','Diadopsi',19),
(20,'Bimo','2019-10-23','Betina','Karantina',20),
(21,'Cleo','2015-09-04','Jantan','Tersedia',7),
(22,'Dodi','2017-12-19','Jantan','Tersedia',8),
(23,'Elmo','2016-12-12','Jantan','Tersedia',1),
(24,'Fifi','2019-08-01','Betina','Karantina',20),
(25,'Gigi','2020-04-08','Betina','Tersedia',8),
(26,'Haru','2018-03-26','Betina','Karantina',15),
(27,'Indy','2015-10-19','Jantan','Diadopsi',19),
(28,'Jojo','2022-06-01','Jantan','Diadopsi',4),
(29,'Kiki','2021-02-08','Jantan','Karantina',5),
(30,'Lola','2017-12-24','Jantan','Tersedia',2),
(31,'Mimi','2016-11-10','Betina','Karantina',19),
(32,'Nana','2018-03-28','Betina','Tersedia',15),
(33,'Oki','2022-09-21','Betina','Karantina',13),
(34,'Pipi','2018-01-20','Betina','Diadopsi',3),
(35,'Qoqo','2021-09-15','Jantan','Diadopsi',11),
(36,'Riri','2021-10-08','Betina','Tersedia',3),
(37,'Sisi','2017-07-26','Jantan','Karantina',9),
(38,'Titi','2021-06-18','Jantan','Tersedia',16),
(39,'Uwu','2020-10-26','Betina','Diadopsi',6),
(40,'Vivi','2021-07-25','Betina','Karantina',16),
(41,'Wiwi','2016-01-09','Betina','Diadopsi',14),
(42,'Xixi','2018-09-27','Betina','Karantina',4),
(43,'Yoyo','2016-10-20','Betina','Diadopsi',16),
(44,'Zizi','2018-03-26','Betina','Karantina',2),
(45,'Aldo','2020-02-06','Jantan','Diadopsi',9),
(46,'Bela','2018-08-17','Jantan','Diadopsi',17),
(47,'Caca','2015-01-05','Betina','Diadopsi',2),
(48,'Dede','2017-02-07','Betina','Karantina',16),
(49,'Edo','2022-01-05','Betina','Tersedia',7),
(50,'Fafa','2017-12-29','Jantan','Diadopsi',15),
(51,'Gaga','2022-10-31','Betina','Tersedia',1),
(52,'Hehe','2022-01-24','Jantan','Karantina',6),
(53,'Ibu','2018-06-26','Jantan','Karantina',14),
(54,'Jaja','2016-01-17','Jantan','Tersedia',15),
(55,'Kaka','2016-04-26','Jantan','Diadopsi',10),
(56,'Lala','2020-09-15','Betina','Diadopsi',16),
(57,'Mama','2020-04-18','Jantan','Diadopsi',18),
(58,'Nono','2016-08-15','Betina','Tersedia',20),
(59,'Otto','2020-09-12','Jantan','Tersedia',9),
(60,'Papa','2023-08-31','Betina','Diadopsi',17),
(61,'Ququ','2017-12-30','Jantan','Diadopsi',10),
(62,'Roro','2021-08-01','Betina','Tersedia',15),
(63,'Soso','2021-01-15','Betina','Diadopsi',11),
(64,'Tata','2021-03-10','Betina','Diadopsi',11),
(65,'Mochi','2017-02-11','Jantan','Karantina',13),
(66,'Oreo','2017-08-14','Betina','Tersedia',11),
(67,'Koko','2023-05-09','Betina','Karantina',13),
(68,'Luna','2019-05-01','Jantan','Diadopsi',2),
(69,'Putih','2016-06-01','Betina','Tersedia',15),
(70,'Hitam','2016-02-13','Betina','Tersedia',5),
(71,'Comel','2019-08-07','Jantan','Tersedia',16),
(72,'Gendut','2023-10-07','Betina','Diadopsi',20),
(73,'Manis','2022-10-08','Betina','Karantina',3),
(74,'Bontot','2018-09-07','Betina','Diadopsi',16),
(75,'Cici','2021-01-25','Jantan','Karantina',3),
(76,'Budi','2017-08-19','Betina','Tersedia',3),
(77,'Dora','2019-11-13','Jantan','Karantina',4),
(78,'Toto','2019-12-23','Jantan','Karantina',10),
(79,'Lili','2015-04-29','Jantan','Diadopsi',2),
(80,'Panda','2018-04-16','Betina','Diadopsi',14),
(81,'Nico','2016-08-19','Jantan','Karantina',14),
(82,'Rara','2021-05-07','Jantan','Tersedia',6),
(83,'Suri','2015-11-20','Betina','Karantina',8),
(84,'Bimo','2020-07-31','Jantan','Tersedia',15),
(85,'Cleo','2022-02-25','Betina','Diadopsi',9),
(86,'Dodi','2022-06-24','Jantan','Diadopsi',10),
(87,'Elmo','2022-08-07','Jantan','Tersedia',15),
(88,'Fifi','2018-11-16','Betina','Karantina',14),
(89,'Gigi','2022-09-28','Betina','Diadopsi',10),
(90,'Haru','2017-03-26','Betina','Diadopsi',4),
(91,'Indy','2017-08-29','Betina','Karantina',12),
(92,'Jojo','2021-06-12','Betina','Karantina',10),
(93,'Kiki','2015-03-31','Betina','Diadopsi',1),
(94,'Lola','2021-05-07','Jantan','Karantina',16),
(95,'Mimi','2018-03-18','Jantan','Karantina',12),
(96,'Nana','2017-06-16','Jantan','Karantina',9),
(97,'Oki','2022-08-08','Jantan','Karantina',4),
(98,'Pipi','2022-01-14','Jantan','Diadopsi',15),
(99,'Qoqo','2015-05-17','Betina','Karantina',5),
(100,'Riri','2016-01-05','Betina','Diadopsi',14),
(101,'Sisi','2016-12-20','Jantan','Tersedia',18),
(102,'Titi','2019-02-07','Betina','Tersedia',9),
(103,'Uwu','2020-05-27','Betina','Karantina',11),
(104,'Vivi','2016-04-16','Betina','Tersedia',5),
(105,'Wiwi','2023-06-17','Jantan','Karantina',13),
(106,'Xixi','2021-04-01','Betina','Tersedia',13),
(107,'Yoyo','2015-02-27','Betina','Karantina',4),
(108,'Zizi','2020-02-06','Betina','Karantina',9),
(109,'Aldo','2021-07-22','Betina','Karantina',12),
(110,'Bela','2016-03-19','Jantan','Diadopsi',1),
(111,'Caca','2021-12-12','Betina','Karantina',8),
(112,'Dede','2022-04-06','Jantan','Karantina',15),
(113,'Edo','2022-11-11','Betina','Karantina',14),
(114,'Fafa','2016-04-22','Jantan','Tersedia',2),
(115,'Gaga','2018-05-31','Betina','Tersedia',4),
(116,'Hehe','2017-08-19','Jantan','Diadopsi',15),
(117,'Ibu','2019-02-28','Betina','Karantina',5),
(118,'Jaja','2019-08-27','Jantan','Diadopsi',20),
(119,'Kaka','2019-07-30','Betina','Tersedia',12),
(120,'Lala','2017-06-08','Betina','Diadopsi',8),
(121,'Mama','2019-01-25','Jantan','Karantina',12),
(122,'Nono','2021-02-08','Betina','Tersedia',13),
(123,'Otto','2018-02-04','Jantan','Tersedia',15),
(124,'Papa','2016-01-11','Jantan','Karantina',20),
(125,'Ququ','2015-03-29','Jantan','Diadopsi',8),
(126,'Roro','2016-05-30','Jantan','Tersedia',18),
(127,'Soso','2017-04-28','Jantan','Tersedia',11),
(128,'Tata','2023-09-07','Jantan','Karantina',1),
(129,'Mochi','2018-02-09','Jantan','Tersedia',18),
(130,'Oreo','2017-10-23','Jantan','Tersedia',1),
(131,'Koko','2016-06-23','Jantan','Diadopsi',8),
(132,'Luna','2021-08-08','Betina','Tersedia',6),
(133,'Putih','2017-12-22','Jantan','Tersedia',14),
(134,'Hitam','2020-11-24','Jantan','Karantina',3),
(135,'Comel','2020-05-04','Betina','Diadopsi',17),
(136,'Gendut','2021-08-28','Jantan','Diadopsi',17),
(137,'Manis','2017-06-26','Jantan','Karantina',17),
(138,'Bontot','2018-05-20','Betina','Karantina',1),
(139,'Cici','2015-09-07','Betina','Diadopsi',14),
(140,'Budi','2022-09-11','Jantan','Diadopsi',15),
(141,'Dora','2015-10-29','Jantan','Diadopsi',20),
(142,'Toto','2016-08-30','Jantan','Tersedia',9),
(143,'Lili','2022-01-01','Betina','Diadopsi',20),
(144,'Panda','2020-12-13','Betina','Diadopsi',17),
(145,'Nico','2021-10-15','Betina','Tersedia',4),
(146,'Rara','2022-05-05','Jantan','Diadopsi',15),
(147,'Suri','2017-07-24','Betina','Diadopsi',15),
(148,'Bimo','2019-06-22','Betina','Karantina',4),
(149,'Cleo','2018-07-04','Betina','Diadopsi',9),
(150,'Dodi','2019-03-14','Jantan','Karantina',16);
/*!40000 ALTER TABLE `hewan` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Temporary table structure for view `hewandenganumur`
--

DROP TABLE IF EXISTS `hewandenganumur`;
/*!50001 DROP VIEW IF EXISTS `hewandenganumur`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8mb4;
/*!50001 CREATE VIEW `hewandenganumur` AS SELECT
 1 AS `id_hewan`,
  1 AS `nama`,
  1 AS `tgl_lahir`,
  1 AS `jenis_kelamin`,
  1 AS `status`,
  1 AS `id_shelter`,
  1 AS `umur` */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `kucing`
--

DROP TABLE IF EXISTS `kucing`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `kucing` (
  `id_hewan` int(11) NOT NULL,
  `kebiasaan` enum('Indoor','Outdoor','Keduanya') NOT NULL,
  PRIMARY KEY (`id_hewan`),
  CONSTRAINT `fk_kucing_hewan` FOREIGN KEY (`id_hewan`) REFERENCES `hewan` (`id_hewan`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `kucing`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `kucing` WRITE;
/*!40000 ALTER TABLE `kucing` DISABLE KEYS */;
INSERT INTO `kucing` VALUES
(1,'Indoor'),
(6,'Outdoor'),
(7,'Indoor'),
(9,'Outdoor'),
(20,'Outdoor'),
(21,'Indoor'),
(23,'Indoor'),
(26,'Outdoor'),
(35,'Outdoor'),
(38,'Outdoor'),
(39,'Keduanya'),
(41,'Outdoor'),
(42,'Outdoor'),
(43,'Keduanya'),
(49,'Keduanya'),
(51,'Keduanya'),
(52,'Indoor'),
(53,'Outdoor'),
(54,'Outdoor'),
(56,'Indoor'),
(57,'Outdoor'),
(59,'Keduanya'),
(60,'Keduanya'),
(62,'Keduanya'),
(63,'Indoor'),
(65,'Keduanya'),
(68,'Outdoor'),
(69,'Indoor'),
(71,'Outdoor'),
(73,'Outdoor'),
(75,'Keduanya'),
(76,'Indoor'),
(88,'Indoor'),
(90,'Outdoor'),
(92,'Outdoor'),
(93,'Keduanya'),
(95,'Indoor'),
(102,'Keduanya'),
(110,'Indoor'),
(113,'Keduanya'),
(114,'Outdoor'),
(115,'Outdoor'),
(121,'Indoor'),
(129,'Keduanya'),
(132,'Keduanya'),
(139,'Outdoor'),
(142,'Indoor'),
(143,'Indoor'),
(144,'Keduanya'),
(147,'Outdoor');
/*!40000 ALTER TABLE `kucing` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `pembayaran`
--

DROP TABLE IF EXISTS `pembayaran`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `pembayaran` (
  `id_pembayaran` int(11) NOT NULL AUTO_INCREMENT,
  `jumlah_bayar` decimal(12,2) NOT NULL CHECK (`jumlah_bayar` >= 0),
  `metode` enum('Transfer Bank','Kartu Kredit','Dompet Digital','Tunai') NOT NULL,
  `waktu` datetime NOT NULL,
  `id_adopsi` int(11) NOT NULL,
  PRIMARY KEY (`id_pembayaran`),
  UNIQUE KEY `id_adopsi` (`id_adopsi`),
  CONSTRAINT `fk_pembayaran_adopsi` FOREIGN KEY (`id_adopsi`) REFERENCES `adopsi` (`id_adopsi`) ON UPDATE CASCADE
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pembayaran`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `pembayaran` WRITE;
/*!40000 ALTER TABLE `pembayaran` DISABLE KEYS */;
INSERT INTO `pembayaran` VALUES
(1,722381.61,'Tunai','2022-08-20 18:08:37',99),
(2,646886.50,'Kartu Kredit','2024-09-06 05:00:06',74),
(3,1664260.12,'Dompet Digital','2024-12-21 12:46:49',34),
(4,541701.38,'Tunai','2021-09-05 12:42:56',84),
(5,1992744.94,'Tunai','2022-12-27 04:07:44',27),
(6,1782336.19,'Transfer Bank','2023-02-12 15:18:28',100),
(7,362088.37,'Tunai','2022-09-03 21:07:52',54),
(8,1940035.93,'Dompet Digital','2021-12-01 05:09:55',80),
(9,287584.64,'Tunai','2022-03-14 01:43:28',82),
(10,1275533.79,'Tunai','2024-08-02 07:42:44',4),
(11,714347.09,'Transfer Bank','2021-01-13 16:52:32',64),
(12,975669.98,'Dompet Digital','2024-10-02 20:20:03',81),
(13,1509514.51,'Tunai','2024-04-01 14:38:39',70),
(14,858687.23,'Transfer Bank','2022-09-15 06:45:55',38),
(15,1226506.96,'Dompet Digital','2021-10-30 09:16:32',83),
(16,1195801.27,'Tunai','2021-04-25 05:27:44',39),
(17,1107659.95,'Dompet Digital','2024-04-02 11:03:03',62),
(18,1685560.37,'Transfer Bank','2023-08-27 04:07:48',32),
(19,431594.10,'Tunai','2021-05-31 04:32:49',52),
(20,871757.01,'Dompet Digital','2022-06-11 18:00:58',85),
(21,1558937.31,'Tunai','2021-11-06 05:24:56',59),
(22,1100140.23,'Transfer Bank','2021-03-12 09:01:53',10),
(23,446665.47,'Tunai','2022-06-02 08:39:26',8),
(24,1134798.88,'Kartu Kredit','2021-08-20 08:54:37',21),
(25,790338.53,'Dompet Digital','2023-08-11 08:50:10',57),
(26,492503.68,'Dompet Digital','2023-08-16 16:48:14',94),
(27,1268704.38,'Tunai','2023-02-28 02:04:24',97),
(28,1475731.65,'Tunai','2022-04-11 17:00:08',60),
(29,1668347.90,'Dompet Digital','2022-07-27 05:22:28',96),
(30,1412851.51,'Dompet Digital','2022-10-10 01:44:48',44),
(31,1634158.67,'Transfer Bank','2024-02-01 18:02:47',19),
(32,1274437.31,'Kartu Kredit','2022-02-17 11:41:31',41),
(33,1498520.20,'Tunai','2023-06-26 11:19:28',69),
(34,506921.71,'Tunai','2023-09-05 03:45:38',45),
(35,465996.27,'Kartu Kredit','2023-11-24 11:20:17',92),
(36,256826.92,'Transfer Bank','2023-12-25 05:18:45',17),
(37,543411.79,'Dompet Digital','2022-11-17 03:06:34',48),
(38,953804.54,'Kartu Kredit','2022-10-04 12:16:07',33),
(39,938679.34,'Tunai','2024-04-18 21:03:41',36),
(40,1771071.88,'Transfer Bank','2024-02-18 02:00:06',7),
(41,1470395.42,'Kartu Kredit','2023-03-17 17:54:42',77),
(42,1209596.19,'Tunai','2023-10-26 17:36:16',16),
(43,1145858.87,'Kartu Kredit','2022-05-03 09:32:40',30),
(44,1027394.36,'Dompet Digital','2023-04-09 04:25:03',78),
(45,1183291.79,'Dompet Digital','2023-11-16 22:12:24',18),
(46,1298137.84,'Kartu Kredit','2021-10-26 16:07:39',29),
(47,702211.51,'Kartu Kredit','2022-04-09 16:51:19',93),
(48,1894673.98,'Dompet Digital','2022-04-08 07:52:30',79),
(49,1468821.69,'Tunai','2023-12-09 06:00:32',61),
(50,770673.34,'Dompet Digital','2022-06-26 20:19:29',91);
/*!40000 ALTER TABLE `pembayaran` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `pengguna`
--

DROP TABLE IF EXISTS `pengguna`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `pengguna` (
  `user_id` int(11) NOT NULL AUTO_INCREMENT,
  `nama` varchar(100) NOT NULL,
  `email` varchar(150) NOT NULL,
  `alamat_jalan` varchar(200) DEFAULT NULL,
  `alamat_kota` varchar(100) DEFAULT NULL,
  `alamat_provinsi` varchar(100) DEFAULT NULL,
  `alamat_negara` varchar(100) DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB AUTO_INCREMENT=301 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `pengguna`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `pengguna` WRITE;
/*!40000 ALTER TABLE `pengguna` DISABLE KEYS */;
INSERT INTO `pengguna` VALUES
(1,'Cahyono Jailani','dongorankunthara@example.org','Jalan Astana Anyar No. 338','Sabang','Kalimantan Selatan','Indonesia'),
(2,'Yessi Ardianto','cprasetya@example.net','Gang Pelajar Pejuang No. 351','Metro','Gorontalo','Indonesia'),
(3,'Michelle Nasyiah, S.IP','malik94@example.org','Gg. Medokan Ayu No. 5','Tarakan','DI Yogyakarta','Indonesia'),
(4,'Sabar Pradipta','wijayantijaga@example.com','Gang Erlangga No. 5','Medan','Nusa Tenggara Timur','Indonesia'),
(5,'Marsudi Hidayanto, M.Ak','umisantoso@example.net','Jl. Rawamangun No. 48','Tangerang Selatan','Maluku Utara','Indonesia'),
(6,'Hasna Haryanto','pradanaatma@example.com','Jalan Suryakencana No. 67','Kota Administrasi Jakarta Selatan','Jawa Barat','Indonesia'),
(7,'Restu Aryani, M.Farm','iriananasyiah@example.com','Jalan HOS. Cokroaminoto No. 7','Tomohon','Bengkulu','Indonesia'),
(8,'Ir. Indra Riyanti, S.Pt','mangunsongratna@example.org','Gang Ahmad Dahlan No. 80','Tangerang Selatan','Kepulauan Riau','Indonesia'),
(9,'Ade Saptono','darmana46@example.org','Gg. KH Amin Jasuta No. 9','Bukittinggi','Papua','Indonesia'),
(10,'dr. Nurul Budiyanto','ghaliyatinuraini@example.org','Gang Moch. Toha No. 391','Tegal','Sumatera Barat','Indonesia'),
(11,'Jaswadi Pudjiastuti, S.E.','nasyidahdwi@example.org','Jalan R.E Martadinata No. 3','Padang','Papua','Indonesia'),
(12,'Darmanto Haryanto','cmaryati@example.org','Jl. Kutai No. 010','Batam','Banten','Indonesia'),
(13,'Jayeng Prabowo, M.Pd','caket83@example.com','Jalan Waringin No. 31','Surabaya','Sulawesi Selatan','Indonesia'),
(14,'Muni Yolanda','slamet67@example.net','Gg. Sadang Serang No. 333','Bogor','Sulawesi Selatan','Indonesia'),
(15,'Puti Maimunah Pudjiastuti, S.IP','kalimhariyah@example.org','Jl. Moch. Toha No. 677','Padang','Bengkulu','Indonesia'),
(16,'Violet Hasanah','prabuadriansyah@example.com','Gang Jakarta No. 430','Tidore Kepulauan','Maluku Utara','Indonesia'),
(17,'Dr. Candrakanta Hakim','hardiansyahlaila@example.com','Jl. Kutisari Selatan No. 361','Sawahlunto','Banten','Indonesia'),
(18,'Drs. Iriana Rajata','andrianitira@example.net','Jalan Pasirkoja No. 24','Lubuklinggau','DI Yogyakarta','Indonesia'),
(19,'Bagya Nasyiah','haryantiputi@example.net','Jalan Sentot Alibasa No. 354','Palembang','Maluku','Indonesia'),
(20,'Bala Mansur, S.T.','widiastutiirma@example.com','Jl. Pasteur No. 93','Gorontalo','Kepulauan Bangka Belitung','Indonesia'),
(21,'Indra Saefullah, S.Gz','dramadan@example.org','Jalan KH Amin Jasuta No. 27','Sabang','Aceh','Indonesia'),
(22,'Dt. Rafid Uwais','hutagalungzahra@example.org','Jalan Cihampelas No. 50','Gorontalo','Kalimantan Utara','Indonesia'),
(23,'R. Gilda Marpaung','thamrinira@example.com','Jalan Jend. Sudirman No. 5','Surakarta','Kalimantan Utara','Indonesia'),
(24,'Dr. Gawati Pangestu, M.M.','galuh16@example.org','Jalan Merdeka No. 365','Batam','Kepulauan Riau','Indonesia'),
(25,'Drs. Julia Dabukke, S.IP','siska86@example.net','Gg. BKR No. 965','Palangkaraya','Sumatera Selatan','Indonesia'),
(26,'Aurora Fujiati','winarnokeisha@example.org','Gg. Bangka Raya No. 595','Tebingtinggi','Maluku','Indonesia'),
(27,'Ida Puspasari, S.E.I','unjani65@example.com','Gang Tebet Barat Dalam No. 2','Padang','Aceh','Indonesia'),
(28,'Hj. Janet Hastuti','kemal36@example.net','Jalan Monginsidi No. 21','Pekanbaru','Nusa Tenggara Barat','Indonesia'),
(29,'Reksa Napitupulu','prakasavanya@example.com','Jl. Pacuan Kuda No. 91','Padangpanjang','Kalimantan Selatan','Indonesia'),
(30,'Marwata Ardianto','zlailasari@example.org','Jalan Jend. Sudirman No. 8','Banjarbaru','Kalimantan Utara','Indonesia'),
(31,'Karta Purwanti, M.Ak','hasanahemas@example.com','Gang M.H Thamrin No. 27','Kota Administrasi Jakarta Selatan','Kalimantan Utara','Indonesia'),
(32,'Citra Suwarno','eva73@example.com','Gang Raya Setiabudhi No. 1','Bukittinggi','Kalimantan Timur','Indonesia'),
(33,'Heryanto Wasita','tmangunsong@example.org','Gang Cikutra Barat No. 669','Tual','Papua Barat','Indonesia'),
(34,'dr. Simon Andriani, M.Pd','wardianto@example.com','Jalan Pasteur No. 70','Makassar','Riau','Indonesia'),
(35,'Dt. Umaya Wijaya','warjimahendra@example.net','Gg. HOS. Cokroaminoto No. 20','Metro','Maluku Utara','Indonesia'),
(36,'dr. Estiawan Andriani, S.T.','upiksimbolon@example.com','Jalan R.E Martadinata No. 531','Tomohon','Bali','Indonesia'),
(37,'Digdaya Mahendra','herudamanik@example.net','Jl. Rawamangun No. 4','Cimahi','Jambi','Indonesia'),
(38,'Wisnu Saragih','luthfikurniawan@example.net','Jalan Ciumbuleuit No. 3','Tanjungpinang','Maluku','Indonesia'),
(39,'dr. Mitra Uwais','pratiwibakda@example.com','Jalan Veteran No. 16','Subulussalam','Sulawesi Tenggara','Indonesia'),
(40,'Ir. Emong Susanti','tedinugroho@example.net','Gang Kutai No. 53','Kota Administrasi Jakarta Utara','Sulawesi Tengah','Indonesia'),
(41,'Harjasa Sirait, S.Ked','wisnupurnawati@example.net','Jalan Rajiman No. 545','Serang','Kepulauan Riau','Indonesia'),
(42,'Najib Wasita','maryatiharjaya@example.org','Gg. H.J Maemunah No. 7','Banda Aceh','DKI Jakarta','Indonesia'),
(43,'Laksana Uwais, M.Kom.','ilyas63@example.com','Gang Setiabudhi No. 7','Lhokseumawe','Kepulauan Bangka Belitung','Indonesia'),
(44,'Paiman Nasyiah','dimasnababan@example.com','Jl. Ronggowarsito No. 98','Bengkulu','Kalimantan Selatan','Indonesia'),
(45,'Nugraha Nasyidah','jaeman52@example.com','Jalan Cikapayang No. 4','Subulussalam','Sumatera Barat','Indonesia'),
(46,'Natalia Farida','budimankarimah@example.com','Jl. Pasir Koja No. 11','Pasuruan','DI Yogyakarta','Indonesia'),
(47,'Chelsea Prayoga','maryadiemin@example.net','Jalan M.T Haryono No. 77','Semarang','Sulawesi Selatan','Indonesia'),
(48,'Cinthia Pranowo, S.Kom','jelita90@example.net','Jl. Moch. Toha No. 812','Padang Sidempuan','Sulawesi Tengah','Indonesia'),
(49,'Tiara Suryono','csiregar@example.org','Jl. Pasirkoja No. 9','Pagaralam','Jambi','Indonesia'),
(50,'R. Belinda Halim, S.E.','salimahmaryadi@example.com','Gang Gedebage Selatan No. 46','Pekalongan','Sumatera Barat','Indonesia'),
(51,'Unjani Sinaga','mhutapea@example.org','Jl. Moch. Ramdan No. 9','Tual','Kepulauan Riau','Indonesia'),
(52,'Ilyas Setiawan, S.E.','natsirwage@example.net','Jalan Waringin No. 65','Surakarta','Gorontalo','Indonesia'),
(53,'Irma Wijayanti','maryantozulkarnain@example.net','Gg. Dipatiukur No. 45','Yogyakarta','Riau','Indonesia'),
(54,'Anastasia Nainggolan','wahyunihalima@example.com','Gang Cikapayang No. 482','Palembang','Sumatera Barat','Indonesia'),
(55,'Sutan Nyana Manullang, S.Pd','dabukkerika@example.net','Jalan Merdeka No. 712','Tasikmalaya','Lampung','Indonesia'),
(56,'Puspa Maulana','radika77@example.org','Jl. Cikutra Timur No. 65','Ambon','Lampung','Indonesia'),
(57,'Paulin Laksmiwati','lili97@example.org','Gang Erlangga No. 33','Kediri','Sulawesi Barat','Indonesia'),
(58,'dr. Harsana Wijayanti, S.E.I','simanjuntakwarsita@example.net','Jl. S. Parman No. 5','Palembang','Gorontalo','Indonesia'),
(59,'Safina Mangunsong','ubudiman@example.org','Gang Sadang Serang No. 6','Magelang','Papua Barat','Indonesia'),
(60,'Puti Gilda Usamah','ratih78@example.org','Jl. Bangka Raya No. 172','Bandar Lampung','Banten','Indonesia'),
(61,'Dalima Prasasta','balidin45@example.com','Jl. Jend. Sudirman No. 969','Payakumbuh','Jawa Tengah','Indonesia'),
(62,'Hamima Widiastuti','virmanuyainah@example.com','Gang Tebet Barat Dalam No. 2','Palembang','Nusa Tenggara Timur','Indonesia'),
(63,'Tgk. Faizah Winarno, M.Kom.','sari64@example.com','Gg. Joyoboyo No. 94','Banda Aceh','Riau','Indonesia'),
(64,'Najwa Suwarno, S.H.','agus90@example.net','Jl. Wonoayu No. 21','Lhokseumawe','Sulawesi Tengah','Indonesia'),
(65,'Bakijan Sihombing','prastutiomar@example.net','Jalan Kiaracondong No. 858','Cimahi','Kepulauan Bangka Belitung','Indonesia'),
(66,'R. Jarwa Hutagalung, S.E.I','vsihombing@example.com','Gang Dipatiukur No. 04','Palu','Papua','Indonesia'),
(67,'Daruna Halimah, S.Ked','laila65@example.org','Jl. KH Amin Jasuta No. 46','Bogor','Banten','Indonesia'),
(68,'Daryani Nasyiah','sihombingazalea@example.org','Gang Soekarno Hatta No. 8','Semarang','Jawa Tengah','Indonesia'),
(69,'drg. Indah Firmansyah','gyolanda@example.com','Gg. Cempaka No. 515','Banjarmasin','Riau','Indonesia'),
(70,'Danuja Yuniar, M.Ak','harjasa17@example.org','Gg. Pacuan Kuda No. 931','Semarang','Kalimantan Tengah','Indonesia'),
(71,'Enteng Maulana','maryadimaulana@example.net','Jalan Gardujati No. 0','Balikpapan','Nusa Tenggara Timur','Indonesia'),
(72,'Eman Habibi','lintangsinaga@example.org','Jalan Cikutra Timur No. 75','Palopo','Sumatera Selatan','Indonesia'),
(73,'Ir. Karma Sihombing, M.M.','parisgunawan@example.net','Gg. Surapati No. 17','Batu','DKI Jakarta','Indonesia'),
(74,'Sarah Usada','olivia21@example.org','Gang Erlangga No. 9','Bengkulu','Jambi','Indonesia'),
(75,'Jessica Hasanah','satyahartati@example.org','Jalan Raya Setiabudhi No. 54','Bukittinggi','Sulawesi Utara','Indonesia'),
(76,'Cinta Wahyudin','iswahyudicawisono@example.org','Gg. Otto Iskandardinata No. 656','Kotamobagu','Maluku','Indonesia'),
(77,'Maryadi Laksmiwati','daniswara98@example.org','Gg. Stasiun Wonokromo No. 14','Pagaralam','Bali','Indonesia'),
(78,'T. Viman Sirait, S.I.Kom','asmianto24@example.net','Jl. Ahmad Yani No. 6','Bontang','Bali','Indonesia'),
(79,'Mutia Siregar','nurainiirfan@example.com','Gang Raya Ujungberung No. 9','Banjarbaru','Jawa Tengah','Indonesia'),
(80,'Empluk Handayani','puputhardiansyah@example.net','Gang Veteran No. 38','Malang','Sulawesi Selatan','Indonesia'),
(81,'Dr. Paramita Widiastuti','suryonoviolet@example.net','Jl. Suniaraja No. 0','Gorontalo','Maluku','Indonesia'),
(82,'Devi Ardianto','endrarajasa@example.com','Jalan Gegerkalong Hilir No. 83','Batu','Riau','Indonesia'),
(83,'Setya Sihotang','fujiatiaris@example.com','Gang PHH. Mustofa No. 605','Solok','Sulawesi Tenggara','Indonesia'),
(84,'Rini Wacana','cecep61@example.org','Gang Sadang Serang No. 521','Pontianak','Sumatera Selatan','Indonesia'),
(85,'Hj. Yulia Maulana, S.H.','nalartampubolon@example.net','Gg. Kiaracondong No. 921','Yogyakarta','Sumatera Barat','Indonesia'),
(86,'Sabrina Pratiwi','wpratiwi@example.net','Jalan Cempaka No. 774','Semarang','Bengkulu','Indonesia'),
(87,'Ani Hakim','cinthia14@example.com','Jalan H.J Maemunah No. 998','Pangkalpinang','Banten','Indonesia'),
(88,'Rahmi Kusmawati','ppertiwi@example.org','Gg. Dr. Djunjunan No. 18','Banjar','Kalimantan Utara','Indonesia'),
(89,'R. Karimah Wahyudin','hfarida@example.net','Gang Sadang Serang No. 9','Sorong','Nusa Tenggara Barat','Indonesia'),
(90,'Harsana Suryono, S.T.','sitompuldacin@example.net','Jalan Indragiri No. 2','Lhokseumawe','Riau','Indonesia'),
(91,'dr. Margana Sitompul','milahutasoit@example.com','Gg. Monginsidi No. 05','Denpasar','Kalimantan Tengah','Indonesia'),
(92,'Bahuwarna Sihombing','amalia33@example.org','Gg. Wonoayu No. 962','Denpasar','Jawa Timur','Indonesia'),
(93,'Najib Prakasa, M.M.','agunawan@example.com','Jalan Jayawijaya No. 473','Tangerang','Sulawesi Tengah','Indonesia'),
(94,'Dr. Hari Marpaung','xsuryono@example.com','Gg. Cikapayang No. 7','Purwokerto','Gorontalo','Indonesia'),
(95,'Mila Mulyani','suryonocaturangga@example.org','Jalan Merdeka No. 50','Banjarbaru','Riau','Indonesia'),
(96,'Alambana Simanjuntak','jarwi61@example.com','Jalan HOS. Cokroaminoto No. 472','Palopo','Aceh','Indonesia'),
(97,'R. Pia Sudiati, M.TI.','budiyantokambali@example.org','Gg. Moch. Toha No. 10','Padangpanjang','Sulawesi Tenggara','Indonesia'),
(98,'Dt. Laswi Mulyani','kusumobetania@example.net','Gg. W.R. Supratman No. 953','Denpasar','DKI Jakarta','Indonesia'),
(99,'Dr. Kadir Damanik, S.Sos','cahya88@example.net','Gg. Soekarno Hatta No. 05','Malang','Kalimantan Utara','Indonesia'),
(100,'Garan Palastri','prastutimulya@example.org','Gg. Ronggowarsito No. 2','Batam','Sulawesi Tenggara','Indonesia'),
(101,'Sutan Timbul Gunarto, S.Psi','sihombinghadi@example.org','Gang Pasir Koja No. 345','Sungai Penuh','Nusa Tenggara Barat','Indonesia'),
(102,'Maida Winarno, S.Psi','drajat56@example.com','Gang Soekarno Hatta No. 4','Tegal','Sulawesi Selatan','Indonesia'),
(103,'T. Galur Jailani','astutinatalia@example.org','Gang Rajawali Timur No. 96','Tanjungbalai','Sulawesi Tengah','Indonesia'),
(104,'Ika Uyainah','tugiman05@example.net','Gg. Siliwangi No. 2','Bontang','Sulawesi Tengah','Indonesia'),
(105,'Ir. Ghani Tarihoran, S.Sos','mahendracahyanto@example.org','Gg. Soekarno Hatta No. 71','Pontianak','Papua','Indonesia'),
(106,'Yosef Latupono','psihotang@example.com','Gg. M.T Haryono No. 81','Kota Administrasi Jakarta Selatan','Sulawesi Tengah','Indonesia'),
(107,'Farah Maheswara','padmasuwarno@example.net','Gg. Jend. Sudirman No. 7','Tangerang Selatan','Maluku Utara','Indonesia'),
(108,'Zizi Manullang','cager88@example.com','Jalan Gedebage Selatan No. 237','Mataram','Papua','Indonesia'),
(109,'Atmaja Saragih','ipuspasari@example.net','Gang Tubagus Ismail No. 1','Lubuklinggau','DI Yogyakarta','Indonesia'),
(110,'Dian Puspasari','rhastuti@example.net','Gg. S. Parman No. 182','Cimahi','Jawa Timur','Indonesia'),
(111,'Puti Hafshah Napitupulu, M.Pd','ihsan58@example.com','Gg. Kendalsari No. 1','Semarang','Jawa Tengah','Indonesia'),
(112,'dr. Zaenab Pratiwi','tiaraoktaviani@example.net','Jl. Cihampelas No. 9','Sungai Penuh','Kalimantan Tengah','Indonesia'),
(113,'Sari Palastri','osafitri@example.org','Gang Pacuan Kuda No. 6','Palu','DI Yogyakarta','Indonesia'),
(114,'Malika Putra','manullangtasnim@example.com','Gg. Raya Setiabudhi No. 5','Blitar','Papua','Indonesia'),
(115,'Emas Hartati','simanjuntakjayadi@example.com','Jl. Cikapayang No. 69','Kota Administrasi Jakarta Barat','Jawa Timur','Indonesia'),
(116,'Oliva Hastuti','kezia53@example.org','Gang Asia Afrika No. 687','Jambi','Nusa Tenggara Timur','Indonesia'),
(117,'Hj. Samiah Halimah','hardiansyahkardi@example.com','Jl. Bangka Raya No. 456','Banjar','Riau','Indonesia'),
(118,'Puspa Suwarno','purnawatijamalia@example.org','Gang M.H Thamrin No. 99','Blitar','Gorontalo','Indonesia'),
(119,'Tgk. Perkasa Megantara','ywahyudin@example.com','Jalan Cihampelas No. 257','Bukittinggi','Sumatera Selatan','Indonesia'),
(120,'Hartana Susanti','marwatasuryatmi@example.com','Gang Ahmad Yani No. 70','Magelang','Jambi','Indonesia'),
(121,'Hj. Aisyah Riyanti, M.Kom.','tpadmasari@example.net','Jl. Indragiri No. 28','Bontang','Kalimantan Utara','Indonesia'),
(122,'Irnanto Sihotang','pertiwidarmaji@example.org','Gg. Laswi No. 89','Ternate','Bengkulu','Indonesia'),
(123,'Salimah Kuswandari','dalimapranowo@example.net','Jalan Erlangga No. 5','Tangerang','Kepulauan Bangka Belitung','Indonesia'),
(124,'T. Darmanto Salahudin','pnababan@example.com','Gg. Raya Setiabudhi No. 58','Kotamobagu','Jambi','Indonesia'),
(125,'Azalea Suwarno','wwidodo@example.org','Gang Suryakencana No. 73','Prabumulih','Kepulauan Riau','Indonesia'),
(126,'R.A. Hesti Hardiansyah','endra19@example.net','Jl. Dipenogoro No. 57','Tangerang Selatan','Jawa Barat','Indonesia'),
(127,'Warji Kusmawati, M.Kom.','pudjiastutigalih@example.com','Gang Peta No. 0','Sungai Penuh','Kalimantan Timur','Indonesia'),
(128,'Raihan Widodo','natsirrafid@example.org','Gg. BKR No. 499','Magelang','Lampung','Indonesia'),
(129,'Diana Mayasari','iyuniar@example.net','Jalan Gardujati No. 27','Tasikmalaya','Kalimantan Barat','Indonesia'),
(130,'Praba Nababan, S.E.','siraitlantar@example.net','Gang Dipatiukur No. 12','Pariaman','Maluku Utara','Indonesia'),
(131,'Ivan Santoso','adinatarajata@example.com','Jalan Pasirkoja No. 5','Banjarmasin','Jambi','Indonesia'),
(132,'Jefri Saragih, S.IP','ksamosir@example.com','Jalan Merdeka No. 727','Padangpanjang','Sulawesi Utara','Indonesia'),
(133,'Puti Paramita Hassanah, M.TI.','jumarizulaika@example.net','Jl. Cikutra Barat No. 8','Samarinda','Jawa Tengah','Indonesia'),
(134,'Maman Putra','marwatanurdiyanti@example.com','Gg. Cikutra Barat No. 28','Palu','Bengkulu','Indonesia'),
(135,'Ayu Widodo, S.T.','zwulandari@example.com','Jl. Cihampelas No. 51','Batam','Sulawesi Utara','Indonesia'),
(136,'R.M. Pranata Yuniar, S.Farm','suwarnorendy@example.org','Gang Sukabumi No. 18','Sungai Penuh','Nusa Tenggara Timur','Indonesia'),
(137,'R.M. Ozy Prasetyo','nugrohogawati@example.com','Gg. Rawamangun No. 8','Prabumulih','Papua Barat','Indonesia'),
(138,'Enteng Nainggolan','jumariharyanti@example.com','Gg. Kendalsari No. 331','Serang','Banten','Indonesia'),
(139,'Cut Hesti Gunawan, S.Sos','rrahimah@example.net','Jl. Cikutra Barat No. 843','Samarinda','Maluku Utara','Indonesia'),
(140,'Sutan Hendri Salahudin','lukitamegantara@example.net','Jl. Medokan Ayu No. 6','Banjarbaru','Nusa Tenggara Timur','Indonesia'),
(141,'Laila Maryadi','unovitasari@example.net','Jalan Jayawijaya No. 33','Tarakan','Jawa Tengah','Indonesia'),
(142,'Puti Nuraini','jhakim@example.com','Jalan Pasir Koja No. 517','Bekasi','Kalimantan Selatan','Indonesia'),
(143,'Ir. Azalea Hidayanto','santosoperkasa@example.net','Gg. Laswi No. 40','Depok','Jambi','Indonesia'),
(144,'Farhunnisa Widodo, S.H.','sprasasta@example.net','Gang Rawamangun No. 5','Kota Administrasi Jakarta Selatan','Papua','Indonesia'),
(145,'Vanya Simanjuntak','ghastuti@example.net','Gang Tubagus Ismail No. 932','Kota Administrasi Jakarta Selatan','Sulawesi Barat','Indonesia'),
(146,'Qori Prasasta','gaduh35@example.org','Gang Rajiman No. 7','Cimahi','Nusa Tenggara Timur','Indonesia'),
(147,'Vero Namaga','estionopalastri@example.com','Jl. Otto Iskandardinata No. 062','Batam','Jawa Tengah','Indonesia'),
(148,'Almira Safitri','pratamacornelia@example.com','Jalan Abdul Muis No. 8','Balikpapan','Bali','Indonesia'),
(149,'Hj. Mala Nugroho, S.IP','gabriella40@example.net','Jl. Kebonjati No. 8','Manado','Kepulauan Riau','Indonesia'),
(150,'Padmi Prasasta','xhutapea@example.com','Jalan Suryakencana No. 0','Padang','Papua Barat','Indonesia'),
(151,'Adika Narpati, S.Pt','epuspasari@example.org','Gang Ahmad Yani No. 803','Tanjungbalai','Sulawesi Utara','Indonesia'),
(152,'Hj. Ami Nugroho','bakiman14@example.net','Jl. Kutai No. 614','Salatiga','Kalimantan Utara','Indonesia'),
(153,'Usyi Mandala','iwastuti@example.com','Jalan Dipenogoro No. 31','Sibolga','Riau','Indonesia'),
(154,'Garda Puspasari','asmuni47@example.com','Jalan Indragiri No. 6','Yogyakarta','Lampung','Indonesia'),
(155,'Harjasa Usada','cakrajiyahutasoit@example.com','Gang Siliwangi No. 104','Mataram','Kepulauan Riau','Indonesia'),
(156,'Melinda Puspita','mustofacayadi@example.org','Jl. Dipatiukur No. 81','Meulaboh','Aceh','Indonesia'),
(157,'Sutan Kenzie Nurdiyanti','raden61@example.net','Jalan HOS. Cokroaminoto No. 44','Bontang','DI Yogyakarta','Indonesia'),
(158,'Novi Hutagalung','tedisaefullah@example.com','Jalan Medokan Ayu No. 99','Palu','Gorontalo','Indonesia'),
(159,'Zaenab Safitri','lulutardianto@example.com','Gang Rumah Sakit No. 92','Madiun','Kalimantan Selatan','Indonesia'),
(160,'Icha Usada','lsihotang@example.com','Gg. Sukabumi No. 65','Salatiga','Kalimantan Selatan','Indonesia'),
(161,'Malika Pranowo, M.M.','wirdapudjiastuti@example.org','Gg. Tebet Barat Dalam No. 24','Kota Administrasi Jakarta Pusat','Jawa Tengah','Indonesia'),
(162,'Dr. Restu Wacana','ewaluyo@example.com','Gg. Jayawijaya No. 1','Yogyakarta','Kalimantan Utara','Indonesia'),
(163,'Murti Yuliarti, S.Sos','ganda90@example.net','Jalan Antapani Lama No. 438','Palopo','Papua','Indonesia'),
(164,'Drs. Eka Situmorang','xhariyah@example.net','Jalan PHH. Mustofa No. 11','Malang','Maluku','Indonesia'),
(165,'Nabila Kuswoyo','viktor38@example.com','Gang Cikutra Timur No. 01','Makassar','Gorontalo','Indonesia'),
(166,'R.M. Gamblang Purwanti','jinawimaryati@example.org','Jalan Surapati No. 2','Subulussalam','Sulawesi Selatan','Indonesia'),
(167,'Ihsan Rahayu','vdabukke@example.net','Jalan Lembong No. 419','Prabumulih','Kalimantan Barat','Indonesia'),
(168,'Salsabila Lestari','martana37@example.org','Gang Kutai No. 557','Tangerang','Maluku','Indonesia'),
(169,'Digdaya Samosir','wulandariembuh@example.com','Gg. Ir. H. Djuanda No. 6','Serang','Jawa Barat','Indonesia'),
(170,'Kawaya Yuliarti, S.E.','langgraini@example.com','Gang HOS. Cokroaminoto No. 579','Manado','Aceh','Indonesia'),
(171,'Dr. Martani Lailasari','mmansur@example.com','Gang Cempaka No. 21','Tual','Kalimantan Timur','Indonesia'),
(172,'Ella Haryanti','atmaja43@example.net','Jl. Pasteur No. 5','Tual','Sulawesi Tengah','Indonesia'),
(173,'Hj. Umi Prayoga, S.Pd','solehjailani@example.net','Gang Sentot Alibasa No. 713','Ternate','Nusa Tenggara Timur','Indonesia'),
(174,'Sutan Rahmat Putra, S.Gz','nmarbun@example.org','Jalan Raya Ujungberung No. 4','Samarinda','Kepulauan Riau','Indonesia'),
(175,'Zelaya Yolanda','mansurridwan@example.org','Jl. Ir. H. Djuanda No. 5','Tual','Riau','Indonesia'),
(176,'Sari Agustina','kmegantara@example.net','Jalan KH Amin Jasuta No. 684','Palu','Sumatera Barat','Indonesia'),
(177,'Siska Widiastuti','lala01@example.net','Gg. Lembong No. 37','Madiun','Lampung','Indonesia'),
(178,'Dr. Winda Siregar, S.Gz','hana92@example.net','Gg. Asia Afrika No. 277','Pagaralam','Kalimantan Selatan','Indonesia'),
(179,'Tgk. Cornelia Yolanda, S.I.Kom','salsabilasihombing@example.org','Gg. Otto Iskandardinata No. 2','Mojokerto','Maluku','Indonesia'),
(180,'Cinta Adriansyah, S.Kom','sidiqyolanda@example.org','Jl. Kutai No. 870','Tanjungpinang','Nusa Tenggara Barat','Indonesia'),
(181,'Belinda Pradana','anggrainizulfa@example.org','Jl. Dr. Djunjunan No. 54','Metro','Jawa Barat','Indonesia'),
(182,'Lurhur Yulianti','cahyonuraini@example.net','Gang Lembong No. 2','Singkawang','Sulawesi Tengah','Indonesia'),
(183,'R. Lanjar Winarno, S.I.Kom','cawukjanuar@example.org','Jalan Pacuan Kuda No. 13','Tanjungbalai','Bali','Indonesia'),
(184,'Pangeran Wijaya','hendra89@example.net','Jl. Yos Sudarso No. 13','Subulussalam','DI Yogyakarta','Indonesia'),
(185,'Zelda Purnawati','tarihoranmahesa@example.org','Jalan Waringin No. 091','Kota Administrasi Jakarta Selatan','Kalimantan Tengah','Indonesia'),
(186,'Sari Mardhiyah','halimraden@example.net','Gang Ciwastra No. 8','Kota Administrasi Jakarta Utara','DKI Jakarta','Indonesia'),
(187,'Tgk. Hartaka Rahmawati','candrakanta83@example.org','Gang Moch. Toha No. 316','Palu','Bengkulu','Indonesia'),
(188,'Dina Anggriawan, S.E.','unjani21@example.org','Gang Jend. A. Yani No. 24','Yogyakarta','Bali','Indonesia'),
(189,'Tania Permata','sidiq48@example.org','Gang Monginsidi No. 07','Medan','Jawa Timur','Indonesia'),
(190,'Oni Adriansyah, S.Gz','danang91@example.org','Jl. Waringin No. 62','Bandung','Sulawesi Barat','Indonesia'),
(191,'Jamalia Saputra, S.Pt','marganaprayoga@example.com','Jl. Jend. Sudirman No. 806','Pangkalpinang','Lampung','Indonesia'),
(192,'Gading Pangestu','marwata18@example.com','Jalan Rawamangun No. 97','Tegal','Kalimantan Utara','Indonesia'),
(193,'Kemba Situmorang','adinatawasita@example.org','Jalan Kapten Muslihat No. 41','Pontianak','Kalimantan Utara','Indonesia'),
(194,'Indah Iswahyudi, S.Gz','dlestari@example.org','Gang Erlangga No. 1','Pekalongan','Jawa Barat','Indonesia'),
(195,'Ibrahim Wijayanti','lhutagalung@example.net','Gg. Pasirkoja No. 65','Palopo','Kepulauan Riau','Indonesia'),
(196,'KH. Lanang Kurniawan, S.E.I','cakrawalapangestu@example.org','Gg. Rawamangun No. 61','Yogyakarta','Kalimantan Tengah','Indonesia'),
(197,'Puti Cornelia Manullang, S.Pd','dsamosir@example.com','Gg. Ir. H. Djuanda No. 356','Tangerang','Maluku','Indonesia'),
(198,'H. Kunthara Prayoga','gantar06@example.com','Jalan HOS. Cokroaminoto No. 003','Dumai','Riau','Indonesia'),
(199,'Yulia Hutagalung','yulia19@example.net','Jl. H.J Maemunah No. 21','Bitung','Banten','Indonesia'),
(200,'Yuni Widodo','khardiansyah@example.net','Gg. W.R. Supratman No. 532','Kota Administrasi Jakarta Utara','Sulawesi Tengah','Indonesia'),
(201,'R.A. Septi Sihombing','bagiyasusanti@example.net','Gang Cikutra Barat No. 462','Dumai','Sumatera Utara','Indonesia'),
(202,'Violet Tamba','rikapalastri@example.org','Gang Ahmad Yani No. 5','Kota Administrasi Jakarta Timur','Kalimantan Barat','Indonesia'),
(203,'Umar Setiawan','simanjuntakmichelle@example.org','Gang Indragiri No. 213','Kota Administrasi Jakarta Timur','Bengkulu','Indonesia'),
(204,'R. Elma Prakasa, M.Farm','kasiyahwibisono@example.com','Jalan Setiabudhi No. 9','Salatiga','Kalimantan Selatan','Indonesia'),
(205,'Ani Iswahyudi','tnamaga@example.org','Jl. Asia Afrika No. 7','Magelang','Kalimantan Selatan','Indonesia'),
(206,'Ajimin Maulana','mamannugroho@example.com','Gg. Ciumbuleuit No. 69','Makassar','Maluku','Indonesia'),
(207,'Sadina Simanjuntak','empluk26@example.net','Jalan Otto Iskandardinata No. 2','Lhokseumawe','Lampung','Indonesia'),
(208,'drg. Zizi Purnawati','garangadriansyah@example.net','Gg. Kendalsari No. 917','Magelang','DI Yogyakarta','Indonesia'),
(209,'Timbul Irawan, S.Psi','marsitosimanjuntak@example.org','Gg. Laswi No. 154','Tebingtinggi','Aceh','Indonesia'),
(210,'Eli Zulkarnain','winarnotaufik@example.net','Gang R.E Martadinata No. 17','Cilegon','Sulawesi Tengah','Indonesia'),
(211,'Hana Jailani','firmansyahlanjar@example.org','Jalan Sukabumi No. 181','Serang','Sulawesi Utara','Indonesia'),
(212,'drg. Endah Habibi','astutisilvia@example.org','Jalan Kebonjati No. 88','Surabaya','Lampung','Indonesia'),
(213,'Drs. Zaenab Lestari','jaswadi10@example.com','Gg. Abdul Muis No. 14','Dumai','Gorontalo','Indonesia'),
(214,'Cindy Kusmawati','putraganep@example.net','Gang Waringin No. 41','Denpasar','Papua','Indonesia'),
(215,'Eluh Farida','luwais@example.com','Jalan Pasteur No. 0','Tomohon','Gorontalo','Indonesia'),
(216,'Usyi Rahmawati','winarsihmalik@example.net','Gg. M.T Haryono No. 87','Parepare','Maluku','Indonesia'),
(217,'Drs. Wirda Pertiwi','widodobagiya@example.net','Gg. Jend. Sudirman No. 332','Pagaralam','DKI Jakarta','Indonesia'),
(218,'Ir. Hana Simanjuntak, S.Pt','opangestu@example.org','Gang M.H Thamrin No. 48','Balikpapan','Sumatera Utara','Indonesia'),
(219,'Kasim Rahmawati','akusumo@example.org','Jl. Peta No. 5','Kota Administrasi Jakarta Selatan','Papua Barat','Indonesia'),
(220,'Cawuk Saefullah','mpertiwi@example.com','Gg. BKR No. 53','Tidore Kepulauan','Sulawesi Utara','Indonesia'),
(221,'Indah Lestari','panca05@example.com','Jl. Rajawali Barat No. 7','Kota Administrasi Jakarta Timur','Sulawesi Barat','Indonesia'),
(222,'drg. Ulya Laksita, M.TI.','nainggolantasdik@example.org','Jl. Cikutra Timur No. 86','Pontianak','Bali','Indonesia'),
(223,'Anggabaya Hasanah','harsanapangestu@example.org','Jalan Pelajar Pejuang No. 78','Subulussalam','Jambi','Indonesia'),
(224,'Dr. Elma Kusmawati','permadipanca@example.com','Jalan Pasirkoja No. 1','Bandar Lampung','Papua Barat','Indonesia'),
(225,'R.A. Septi Prasetyo, S.Kom','marpaungkani@example.org','Jalan Antapani Lama No. 22','Jayapura','Kepulauan Bangka Belitung','Indonesia'),
(226,'dr. Michelle Ramadan, S.Kom','syahrini46@example.com','Gang Indragiri No. 085','Langsa','Gorontalo','Indonesia'),
(227,'Gamani Saefullah, S.T.','vharyanto@example.com','Jl. M.T Haryono No. 096','Tangerang Selatan','Jawa Barat','Indonesia'),
(228,'Lili Waskita','yolandaalmira@example.com','Jl. Sadang Serang No. 3','Surakarta','Nusa Tenggara Timur','Indonesia'),
(229,'Jarwi Mahendra','ganep82@example.com','Gg. Suryakencana No. 353','Padangpanjang','Nusa Tenggara Barat','Indonesia'),
(230,'Puti Nadine Habibi, M.Farm','karma94@example.com','Jalan S. Parman No. 684','Banjarbaru','Jawa Tengah','Indonesia'),
(231,'Dr. Dinda Sihotang','himawansiregar@example.com','Gg. Rumah Sakit No. 2','Subulussalam','Bengkulu','Indonesia'),
(232,'KH. Cawisadi Permata, S.E.','hartatikuncara@example.org','Jl. Kendalsari No. 66','Pariaman','Papua Barat','Indonesia'),
(233,'Kasiyah Budiman, S.Psi','firgantorofarhunnisa@example.org','Jl. Medokan Ayu No. 3','Tanjungbalai','Nusa Tenggara Timur','Indonesia'),
(234,'Tgk. Eli Haryanto','eka16@example.net','Gang Erlangga No. 8','Palangkaraya','Jawa Barat','Indonesia'),
(235,'dr. Ajiman Pradana, S.H.','ana53@example.org','Jl. Kutai No. 2','Mataram','Banten','Indonesia'),
(236,'Cut Rina Hardiansyah','quyainah@example.net','Jl. Gardujati No. 76','Purwokerto','Kepulauan Bangka Belitung','Indonesia'),
(237,'Lasmono Anggriawan, S.Pd','nmangunsong@example.net','Gg. Wonoayu No. 2','Bima','Sumatera Selatan','Indonesia'),
(238,'Faizah Sudiati','suryonodarsirah@example.org','Jalan Setiabudhi No. 6','Cilegon','Banten','Indonesia'),
(239,'Gilda Adriansyah','spranowo@example.com','Jl. Joyoboyo No. 7','Bekasi','Sumatera Selatan','Indonesia'),
(240,'Jaga Sihombing','januarkambali@example.net','Jalan Dipenogoro No. 045','Pagaralam','Jambi','Indonesia'),
(241,'Marsudi Narpati, S.Psi','gunawanajiman@example.org','Gang Dipatiukur No. 52','Padang','Riau','Indonesia'),
(242,'Dr. Wirda Andriani, S.E.','ysaputra@example.net','Jl. Jamika No. 697','Denpasar','Papua Barat','Indonesia'),
(243,'Carla Zulkarnain','padma02@example.com','Jl. Pasirkoja No. 5','Tarakan','Sumatera Utara','Indonesia'),
(244,'H. Naradi Safitri','ulvasirait@example.net','Jalan Ronggowarsito No. 62','Balikpapan','Jawa Timur','Indonesia'),
(245,'drg. Galur Permadi, S.Psi','klailasari@example.org','Jl. M.H Thamrin No. 6','Kendari','DKI Jakarta','Indonesia'),
(246,'Clara Mandasari','zhandayani@example.net','Jalan Yos Sudarso No. 0','Padang','Sulawesi Barat','Indonesia'),
(247,'Puti Widya Suartini, M.Ak','puspitahasna@example.org','Gg. Sadang Serang No. 03','Magelang','Sulawesi Utara','Indonesia'),
(248,'Ade Siregar','utama11@example.net','Gang Dr. Djunjunan No. 272','Bima','Kepulauan Bangka Belitung','Indonesia'),
(249,'Jagapati Nainggolan','harimurtilazuardi@example.com','Gg. Cihampelas No. 9','Yogyakarta','Sulawesi Tengah','Indonesia'),
(250,'Yoga Mandasari','lwahyuni@example.net','Gg. Indragiri No. 82','Tegal','Jawa Barat','Indonesia'),
(251,'dr. Ophelia Winarno','wibisonotira@example.net','Gg. Suniaraja No. 47','Bogor','Sulawesi Barat','Indonesia'),
(252,'Daliono Pranowo','hana58@example.net','Jalan Suryakencana No. 885','Tual','Nusa Tenggara Timur','Indonesia'),
(253,'Lidya Wulandari, M.Pd','kpadmasari@example.net','Jalan M.T Haryono No. 062','Surakarta','Bengkulu','Indonesia'),
(254,'Dwi Maulana, S.Gz','kasusramarpaung@example.com','Jl. Setiabudhi No. 19','Tidore Kepulauan','Papua Barat','Indonesia'),
(255,'Asirwada Wibowo, S.Sos','calistasuwarno@example.net','Jalan Antapani Lama No. 4','Solok','Sulawesi Tengah','Indonesia'),
(256,'Drs. Kenes Suartini, S.Kom','rahmiyuliarti@example.com','Gang Jend. Sudirman No. 1','Cirebon','Sumatera Selatan','Indonesia'),
(257,'Puti Saptono','waluyobajragin@example.com','Gg. Abdul Muis No. 72','Bukittinggi','Maluku Utara','Indonesia'),
(258,'Nurul Pertiwi','utamipatricia@example.net','Gg. Ahmad Dahlan No. 281','Padangpanjang','Riau','Indonesia'),
(259,'Gamblang Sitompul, S.I.Kom','tampubolonilyas@example.org','Gg. Stasiun Wonokromo No. 0','Surakarta','Nusa Tenggara Timur','Indonesia'),
(260,'R.A. Azalea Zulaika','ppermadi@example.net','Jalan Sukajadi No. 84','Pematangsiantar','Papua','Indonesia'),
(261,'Dadi Sinaga, M.TI.','claksita@example.com','Jl. Jayawijaya No. 0','Gorontalo','Kalimantan Tengah','Indonesia'),
(262,'drg. Septi Hidayanto, S.E.I','njailani@example.net','Gang BKR No. 102','Sabang','DI Yogyakarta','Indonesia'),
(263,'Siti Waskita','karimanwasita@example.com','Jalan Sukajadi No. 1','Sukabumi','DI Yogyakarta','Indonesia'),
(264,'Elvin Sihombing','hidayantonasrullah@example.net','Gang Sukabumi No. 07','Prabumulih','Kalimantan Timur','Indonesia'),
(265,'Cakrabirawa Salahudin, S.I.Kom','purwapalastri@example.com','Jl. Cihampelas No. 996','Lhokseumawe','Kalimantan Tengah','Indonesia'),
(266,'Cengkal Prasetya','indra68@example.org','Jl. K.H. Wahid Hasyim No. 9','Tomohon','Sulawesi Tenggara','Indonesia'),
(267,'Cinthia Napitupulu','siregarusyi@example.org','Gang Gegerkalong Hilir No. 3','Kotamobagu','DI Yogyakarta','Indonesia'),
(268,'Raisa Wahyuni','makaralaksita@example.com','Gg. Raya Setiabudhi No. 20','Kota Administrasi Jakarta Barat','Kalimantan Timur','Indonesia'),
(269,'Oskar Sitompul, M.Kom.','xsaefullah@example.org','Gang Rajawali Timur No. 81','Sukabumi','DKI Jakarta','Indonesia'),
(270,'Sari Hutagalung, S.Pt','estiono83@example.com','Gg. Kutai No. 084','Sungai Penuh','Jawa Tengah','Indonesia'),
(271,'Tgk. Diana Hariyah, M.TI.','purwanto39@example.net','Gang S. Parman No. 392','Palopo','Jawa Timur','Indonesia'),
(272,'R. Unjani Laksmiwati, S.I.Kom','julipratiwi@example.net','Gg. Moch. Ramdan No. 453','Surakarta','Kalimantan Tengah','Indonesia'),
(273,'Martaka Sinaga','prayoga77@example.org','Gang Indragiri No. 9','Metro','Papua Barat','Indonesia'),
(274,'Genta Maryati','lili18@example.com','Gang Rajiman No. 90','Palembang','Jawa Timur','Indonesia'),
(275,'Gabriella Maryadi','candramulyani@example.net','Gg. Rungkut Industri No. 1','Padangpanjang','Nusa Tenggara Barat','Indonesia'),
(276,'Fathonah Rahayu','laksmiwativera@example.com','Jalan Pacuan Kuda No. 299','Bogor','Sumatera Selatan','Indonesia'),
(277,'Eka Najmudin','pertiwiemil@example.org','Gg. Ciwastra No. 797','Cimahi','Kalimantan Timur','Indonesia'),
(278,'R. Sarah Wahyuni, M.TI.','jrahimah@example.net','Gg. Erlangga No. 9','Kediri','Papua Barat','Indonesia'),
(279,'Karman Mayasari','gmustofa@example.org','Jl. Jend. Sudirman No. 99','Prabumulih','Sumatera Selatan','Indonesia'),
(280,'Umay Suartini','iswahyudiyoga@example.net','Gang Soekarno Hatta No. 863','Yogyakarta','Papua Barat','Indonesia'),
(281,'Dodo Kuswandari','maria91@example.com','Gg. Kutisari Selatan No. 6','Depok','Aceh','Indonesia'),
(282,'Dt. Harimurti Yolanda','permataelisa@example.org','Gang Siliwangi No. 269','Kota Administrasi Jakarta Selatan','Sulawesi Selatan','Indonesia'),
(283,'Puspa Puspita','padma46@example.org','Gg. Stasiun Wonokromo No. 7','Lhokseumawe','Nusa Tenggara Timur','Indonesia'),
(284,'Ghaliyati Hassanah, S.E.','qmandala@example.org','Gg. Sadang Serang No. 9','Cilegon','Papua Barat','Indonesia'),
(285,'drg. Clara Haryanto','iswahyudielvin@example.com','Gg. Ciwastra No. 0','Padang','Sumatera Utara','Indonesia'),
(286,'Maida Anggriawan','michelle02@example.net','Jalan BKR No. 8','Singkawang','Jawa Timur','Indonesia'),
(287,'Candrakanta Irawan','tarihoranjati@example.net','Gang Otto Iskandardinata No. 69','Pematangsiantar','Aceh','Indonesia'),
(288,'Yance Purwanti, M.Farm','iswahyudihasna@example.com','Gang Merdeka No. 3','Padang','Aceh','Indonesia'),
(289,'R. Jefri Sihombing','gara68@example.org','Gg. Jayawijaya No. 4','Metro','Sulawesi Utara','Indonesia'),
(290,'Rini Dabukke','mustikaharyanti@example.net','Gg. Cikutra Timur No. 33','Padang Sidempuan','Nusa Tenggara Timur','Indonesia'),
(291,'Heru Wulandari','saptonojamal@example.com','Jl. Waringin No. 53','Denpasar','Kalimantan Selatan','Indonesia'),
(292,'Teddy Saptono','ppangestu@example.net','Jl. Pacuan Kuda No. 5','Bima','Jambi','Indonesia'),
(293,'Cut Vanesa Namaga','wastutitari@example.com','Gang Wonoayu No. 79','Pasuruan','Kalimantan Barat','Indonesia'),
(294,'Karsa Anggraini','raden39@example.org','Gg. Suniaraja No. 2','Bitung','Sumatera Utara','Indonesia'),
(295,'Mursita Pranowo','ssihotang@example.org','Gg. Siliwangi No. 4','Purwokerto','Kepulauan Bangka Belitung','Indonesia'),
(296,'Karimah Nashiruddin','kawayahalimah@example.net','Jalan Veteran No. 5','Padang','Kepulauan Riau','Indonesia'),
(297,'Hj. Ghaliyati Januar','knajmudin@example.org','Gg. Sukabumi No. 227','Cirebon','Sulawesi Barat','Indonesia'),
(298,'H. Galih Laksita','omansur@example.net','Jalan Suniaraja No. 203','Parepare','Gorontalo','Indonesia'),
(299,'Putri Puspita','qsitorus@example.com','Jalan Tubagus Ismail No. 593','Kotamobagu','Nusa Tenggara Timur','Indonesia'),
(300,'Cindy Wibowo','vanesa87@example.net','Jalan Jend. A. Yani No. 5','Sorong','Riau','Indonesia');
/*!40000 ALTER TABLE `pengguna` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `shelter`
--

DROP TABLE IF EXISTS `shelter`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `shelter` (
  `id_shelter` int(11) NOT NULL AUTO_INCREMENT,
  `nama` varchar(100) NOT NULL,
  `lokasi_kota` varchar(100) NOT NULL,
  `lokasi_negara` varchar(100) NOT NULL,
  PRIMARY KEY (`id_shelter`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `shelter`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `shelter` WRITE;
/*!40000 ALTER TABLE `shelter` DISABLE KEYS */;
INSERT INTO `shelter` VALUES
(1,'Shelter Bandung','Bandung','Indonesia'),
(2,'Shelter Jakarta','Jakarta','Indonesia'),
(3,'Shelter Surabaya','Surabaya','Indonesia'),
(4,'Shelter Yogyakarta','Yogyakarta','Indonesia'),
(5,'Shelter Medan','Medan','Indonesia'),
(6,'Shelter Makassar','Makassar','Indonesia'),
(7,'Shelter Semarang','Semarang','Indonesia'),
(8,'Shelter Palembang','Palembang','Indonesia'),
(9,'Shelter Denpasar','Denpasar','Indonesia'),
(10,'Shelter Malang','Malang','Indonesia'),
(11,'Shelter Bogor','Bogor','Indonesia'),
(12,'Shelter Tangerang','Tangerang','Indonesia'),
(13,'Shelter Depok','Depok','Indonesia'),
(14,'Shelter Bekasi','Bekasi','Indonesia'),
(15,'Shelter Batam','Batam','Indonesia'),
(16,'Shelter Pekanbaru','Pekanbaru','Indonesia'),
(17,'Shelter Balikpapan','Balikpapan','Indonesia'),
(18,'Shelter Manado','Manado','Indonesia'),
(19,'Shelter Ambon','Ambon','Indonesia'),
(20,'Shelter Kupang','Kupang','Indonesia');
/*!40000 ALTER TABLE `shelter` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `stafshelter`
--

DROP TABLE IF EXISTS `stafshelter`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `stafshelter` (
  `user_id` int(11) NOT NULL,
  `id_pegawai` varchar(50) NOT NULL,
  `id_shelter` int(11) NOT NULL,
  PRIMARY KEY (`user_id`),
  UNIQUE KEY `id_pegawai` (`id_pegawai`),
  KEY `fk_staf_shelter` (`id_shelter`),
  CONSTRAINT `fk_staf_shelter` FOREIGN KEY (`id_shelter`) REFERENCES `shelter` (`id_shelter`) ON UPDATE CASCADE,
  CONSTRAINT `fk_staf_user` FOREIGN KEY (`user_id`) REFERENCES `pengguna` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `stafshelter`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `stafshelter` WRITE;
/*!40000 ALTER TABLE `stafshelter` DISABLE KEYS */;
INSERT INTO `stafshelter` VALUES
(3,'STF-0022',14),
(4,'STF-0080',16),
(5,'STF-0075',14),
(6,'STF-0068',9),
(7,'STF-0018',19),
(9,'STF-0044',19),
(11,'STF-0036',6),
(16,'STF-0073',3),
(23,'STF-0054',20),
(24,'STF-0069',18),
(25,'STF-0087',8),
(26,'STF-0012',19),
(27,'STF-0005',1),
(29,'STF-0072',7),
(32,'STF-0082',1),
(33,'STF-0066',9),
(34,'STF-0049',9),
(38,'STF-0078',8),
(41,'STF-0089',19),
(44,'STF-0083',3),
(45,'STF-0077',18),
(47,'STF-0042',15),
(50,'STF-0051',8),
(58,'STF-0002',10),
(61,'STF-0040',18),
(65,'STF-0053',19),
(69,'STF-0094',12),
(75,'STF-0013',8),
(79,'STF-0004',19),
(86,'STF-0030',4),
(87,'STF-0071',17),
(93,'STF-0090',12),
(95,'STF-0085',8),
(100,'STF-0065',11),
(102,'STF-0055',13),
(106,'STF-0033',15),
(109,'STF-0063',12),
(110,'STF-0088',10),
(119,'STF-0024',3),
(120,'STF-0096',18),
(123,'STF-0081',15),
(127,'STF-0057',1),
(129,'STF-0008',13),
(130,'STF-0086',13),
(132,'STF-0038',9),
(136,'STF-0070',1),
(139,'STF-0023',12),
(140,'STF-0099',15),
(144,'STF-0098',12),
(145,'STF-0074',8),
(146,'STF-0020',12),
(147,'STF-0093',17),
(150,'STF-0015',10),
(151,'STF-0045',9),
(155,'STF-0039',20),
(159,'STF-0007',19),
(161,'STF-0006',10),
(162,'STF-0014',4),
(164,'STF-0050',15),
(167,'STF-0034',5),
(169,'STF-0100',9),
(175,'STF-0059',11),
(178,'STF-0052',15),
(179,'STF-0010',7),
(181,'STF-0067',20),
(183,'STF-0076',16),
(186,'STF-0021',18),
(191,'STF-0031',14),
(199,'STF-0091',16),
(208,'STF-0043',14),
(210,'STF-0027',1),
(213,'STF-0009',13),
(214,'STF-0028',14),
(219,'STF-0060',6),
(220,'STF-0047',8),
(226,'STF-0019',2),
(228,'STF-0016',20),
(231,'STF-0046',11),
(233,'STF-0062',7),
(234,'STF-0029',16),
(237,'STF-0048',3),
(240,'STF-0079',16),
(243,'STF-0092',18),
(249,'STF-0061',16),
(260,'STF-0035',14),
(261,'STF-0032',12),
(262,'STF-0095',14),
(266,'STF-0064',9),
(267,'STF-0037',17),
(269,'STF-0001',6),
(270,'STF-0017',4),
(272,'STF-0003',4),
(279,'STF-0025',17),
(281,'STF-0026',11),
(287,'STF-0041',16),
(291,'STF-0097',11),
(293,'STF-0011',3),
(294,'STF-0084',10),
(297,'STF-0058',16),
(298,'STF-0056',11);
/*!40000 ALTER TABLE `stafshelter` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `teleponpengguna`
--

DROP TABLE IF EXISTS `teleponpengguna`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `teleponpengguna` (
  `user_id` int(11) NOT NULL,
  `no_telp` varchar(20) NOT NULL,
  PRIMARY KEY (`user_id`,`no_telp`),
  CONSTRAINT `fk_telp_user` FOREIGN KEY (`user_id`) REFERENCES `pengguna` (`user_id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `teleponpengguna`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `teleponpengguna` WRITE;
/*!40000 ALTER TABLE `teleponpengguna` DISABLE KEYS */;
INSERT INTO `teleponpengguna` VALUES
(1,'+62 (012) 535 5449'),
(2,'(047) 598 7622'),
(3,'+62 (078) 006 8114'),
(3,'+62-23-413-2611'),
(4,'+62 (16) 433 4722'),
(5,'082 005 5778'),
(6,'+62 (0872) 829 9796'),
(7,'088 457 8317'),
(8,'+62 (0564) 240-0825'),
(9,'+62 (0780) 774-0649'),
(9,'+62-0085-368-6632'),
(10,'(063) 416 3894'),
(11,'+62 (0846) 312-2417'),
(12,'(0497) 350 2080'),
(13,'+62 (14) 427-0312'),
(14,'+62-73-564-7570'),
(15,'+62 (120) 461 8723'),
(16,'+62 (0601) 367 3037'),
(17,'+62 (083) 764-5208'),
(17,'081 474 3503'),
(18,'+62 (036) 502-4916'),
(19,'(0195) 005-2812'),
(19,'+62-0009-792-5132'),
(20,'(0358) 573-0652'),
(20,'+62 (0077) 848 7392'),
(21,'0855749821'),
(22,'082 744 8570'),
(23,'+62 (086) 387 8294'),
(23,'086 209 5935'),
(24,'+62 (651) 181 0220'),
(24,'+62-173-109-6789'),
(25,'+62 (0053) 910 3717'),
(25,'+62-058-650-9480'),
(26,'+62 (041) 381 1386'),
(27,'+62-383-928-8619'),
(28,'+62 (761) 228-6852'),
(28,'0897558556'),
(29,'(059) 398-3684'),
(30,'+62 (776) 004-9082'),
(31,'+62 (0043) 350-4888'),
(31,'+62 (10) 752 2459'),
(32,'+62-0344-196-9315'),
(33,'(0279) 194-0961'),
(33,'+62-055-661-3906'),
(34,'+62-0206-799-8136'),
(34,'+62-048-058-9258'),
(35,'+62 (0168) 622 7837'),
(35,'+62 (0622) 517-9086'),
(36,'(093) 898 3314'),
(37,'(059) 116-6553'),
(37,'+62 (590) 425 9792'),
(38,'(0033) 521 1039'),
(39,'+62 (40) 514 7705'),
(39,'+62-833-884-8792'),
(40,'+62-69-804-2043'),
(41,'(036) 815 7981'),
(41,'+62 (0070) 164 5104'),
(42,'+62 (38) 780-0384'),
(42,'+62 (762) 055 5438'),
(43,'+62 (0046) 084 3256'),
(44,'+62 (16) 893 6392'),
(45,'(049) 578-2329'),
(46,'0832482629'),
(47,'+62 (51) 881-2520'),
(47,'+62 (664) 742 3122'),
(48,'+62-99-499-9393'),
(49,'+62-0051-728-9752'),
(50,'(077) 223 3060'),
(51,'(0366) 096 5699'),
(51,'+62 (0633) 312 7975'),
(52,'088 608 6803'),
(52,'0880817703'),
(53,'+62 (0764) 472-6490'),
(53,'082 771 8129'),
(54,'+62-714-521-1175'),
(54,'0826657027'),
(55,'+62 (086) 892 6860'),
(56,'+62 (0742) 123-0940'),
(56,'+62 (59) 222-2064'),
(57,'(0671) 990 3849'),
(57,'(0778) 097 3397'),
(58,'0830220502'),
(59,'+62 (49) 998-2795'),
(59,'+62-099-451-0162'),
(60,'+62-014-556-1681'),
(61,'+62 (0721) 458-6833'),
(62,'+62-019-817-9469'),
(63,'+62 (489) 952 9978'),
(64,'(0729) 343-0107'),
(64,'+62 (039) 660-5583'),
(65,'+62-15-569-9023'),
(65,'+62-34-414-4885'),
(66,'(0608) 495 8803'),
(66,'+62 (577) 027 2785'),
(67,'+62 (512) 405 6829'),
(68,'(029) 145 0461'),
(68,'(0357) 438-4215'),
(69,'+62-031-780-7823'),
(70,'+62 (784) 521 8230'),
(71,'(0681) 040 9157'),
(72,'+62-018-409-4353'),
(72,'+62-448-571-4224'),
(73,'+62 (009) 881-3858'),
(73,'+62 (424) 932 1495'),
(74,'+62-016-629-9795'),
(74,'0870207016'),
(75,'+62 (0538) 179-2171'),
(76,'(0477) 332 9456'),
(77,'(0724) 599-2816'),
(77,'+62 (55) 770-1715'),
(78,'+62 (058) 148-7696'),
(79,'(069) 583-2197'),
(79,'+62-0938-276-1539'),
(80,'(019) 196 5701'),
(80,'+62 (099) 758-1724'),
(81,'(0335) 494 6519'),
(81,'+62-290-227-8469'),
(82,'+62-98-624-0580'),
(83,'+62 (666) 683 2723'),
(83,'+62-85-140-3696'),
(84,'0865624220'),
(85,'+62 (002) 762 1267'),
(86,'(092) 786-8765'),
(86,'+62-0269-097-3846'),
(87,'(0128) 478-1891'),
(87,'0895009751'),
(88,'(0787) 956-5442'),
(88,'+62-0718-949-0580'),
(89,'+62-049-047-1281'),
(89,'+62-094-940-7929'),
(90,'+62 (0429) 444 0192'),
(91,'+62 (75) 243 7280'),
(92,'+62 (129) 276 6372'),
(92,'+62-950-410-2767'),
(93,'+62 (0783) 667-4286'),
(94,'+62-47-658-0185'),
(95,'(094) 544-1496'),
(96,'(0393) 852 5800'),
(97,'+62 (006) 181 7534'),
(98,'+62 (019) 269-2124'),
(98,'+62 (0300) 910 9391'),
(99,'+62 (0717) 465-4214'),
(100,'(0396) 271-2863'),
(100,'+62-066-504-4271'),
(101,'+62 (074) 101-1947'),
(101,'+62-72-006-4256'),
(102,'(071) 985 5460'),
(102,'+62-87-766-1830'),
(103,'+62 (058) 517 4328'),
(103,'+62-063-461-5717'),
(104,'+62-203-093-7618'),
(105,'+62 (0787) 005-0506'),
(106,'+62 (038) 782-9310'),
(106,'+62 (70) 790 3202'),
(107,'(0993) 136-5651'),
(107,'+62-041-792-9111'),
(108,'0825309251'),
(109,'+62 (042) 625-8273'),
(109,'+62-021-962-0710'),
(110,'+62 (0876) 329 0927'),
(110,'+62-21-946-7232'),
(111,'+62 (045) 608 3318'),
(112,'+62 (095) 883-7340'),
(112,'+62-127-722-0054'),
(113,'+62-645-970-1082'),
(114,'+62 (07) 811-3799'),
(114,'+62 (635) 315 3136'),
(115,'+62-631-797-0623'),
(116,'(0265) 273 5435'),
(117,'+62 (09) 042 1776'),
(117,'+62 (53) 492-9298'),
(118,'+62-0168-739-7740'),
(119,'(033) 569-3929'),
(120,'(0177) 318-0632'),
(120,'+62-60-078-0686'),
(121,'+62 (431) 074-2799'),
(122,'+62-77-262-5701'),
(123,'(058) 046 4989'),
(123,'+62-195-085-0165'),
(124,'(0905) 528 1832'),
(124,'+62 (381) 793 3554'),
(125,'086 401 5577'),
(126,'088 178 8479'),
(127,'+62 (137) 116-2411'),
(127,'+62 (41) 272-3804'),
(128,'(082) 756-7043'),
(128,'+62 (003) 856 9557'),
(129,'088 774 9242'),
(130,'+62 (0375) 402 5472'),
(131,'+62 (84) 222-2310'),
(132,'+62 (927) 452 3805'),
(133,'+62 (257) 828-4106'),
(134,'(0275) 208-8547'),
(134,'+62 (0929) 751-7028'),
(135,'+62 (040) 693-7515'),
(136,'+62-093-983-4301'),
(137,'0871863759'),
(138,'+62-45-547-8137'),
(138,'080 562 6565'),
(139,'(074) 266-2991'),
(140,'+62 (34) 785 1961'),
(140,'+62-64-628-7923'),
(141,'+62 (64) 870-4947'),
(141,'0873002447'),
(142,'+62 (046) 273-0147'),
(143,'+62 (095) 979 6471'),
(144,'+62 (52) 045 6504'),
(144,'+62 (890) 429-0257'),
(145,'+62 (0818) 321-9285'),
(145,'+62 (445) 189 5202'),
(146,'+62 (83) 389 8881'),
(146,'084 576 5213'),
(147,'(071) 120-7545'),
(147,'+62 (79) 852 4013'),
(148,'+62 (007) 603 1089'),
(148,'+62 (23) 423 2161'),
(149,'+62-071-401-4011'),
(150,'+62-932-501-2495'),
(151,'+62 (289) 389-1190'),
(152,'083 342 8013'),
(153,'(063) 836 1517'),
(153,'087 592 7125'),
(154,'+62 (082) 754-8015'),
(155,'(0164) 794-7386'),
(156,'+62 (0631) 752-0716'),
(157,'(0811) 266 9121'),
(158,'+62 (353) 961-3674'),
(159,'+62 (0657) 967 8427'),
(160,'+62 (014) 773-2010'),
(161,'+62 (070) 269 5249'),
(162,'+62 (018) 773-4778'),
(163,'+62 (0858) 800 6167'),
(163,'+62 (677) 002 8427'),
(164,'(024) 671-9788'),
(165,'(0782) 806-4796'),
(166,'+62 (450) 090-2559'),
(166,'+62-56-165-5650'),
(167,'+62-0334-911-4152'),
(167,'+62-85-076-9864'),
(168,'+62-79-056-9659'),
(169,'+62 (071) 357-8529'),
(170,'+62 (0563) 718-3440'),
(170,'+62 (0963) 919-1954'),
(171,'+62 (098) 763 5148'),
(172,'+62 (0685) 366-4987'),
(172,'0820829737'),
(173,'+62 (0896) 908 3572'),
(173,'+62-0246-651-3644'),
(174,'+62 (49) 363-1230'),
(175,'+62-0477-866-0324'),
(176,'+62 (04) 482 0700'),
(177,'+62 (059) 416 5162'),
(177,'+62-0198-189-8471'),
(178,'+62 (023) 476-4907'),
(178,'+62-0915-596-3632'),
(179,'+62 (57) 540 9003'),
(179,'0851586734'),
(180,'(044) 134 3275'),
(180,'+62 (0864) 518 3760'),
(181,'(004) 177-6561'),
(181,'+62 (48) 156-9857'),
(182,'+62-02-720-6345'),
(183,'+62 (012) 382 9657'),
(184,'(0242) 954-3165'),
(185,'+62 (073) 420 6969'),
(185,'+62 (66) 930-0893'),
(186,'(0133) 344-7626'),
(186,'+62 (0215) 291 0867'),
(187,'+62-934-362-6204'),
(188,'+62 (0397) 359 7107'),
(189,'+62 (56) 366 9411'),
(190,'0800537599'),
(191,'+62-004-896-4652'),
(191,'0809242875'),
(192,'(0215) 862 4150'),
(193,'+62 (625) 728 7032'),
(193,'+62 (74) 564 4948'),
(194,'+62-032-074-2761'),
(195,'(0075) 877-7121'),
(195,'+62 (134) 228-4993'),
(196,'+62-41-659-1520'),
(196,'+62-89-261-3715'),
(197,'+62 (077) 749 0509'),
(198,'(0592) 304 5018'),
(199,'(016) 423 6103'),
(199,'+62 (66) 421-4764'),
(200,'+62 (51) 583 7767'),
(201,'+62 (0996) 165 9266'),
(202,'+62 (008) 573 2788'),
(203,'+62 (047) 617 4224'),
(204,'+62-0864-908-1081'),
(205,'086 617 9454'),
(206,'(045) 977-9452'),
(206,'+62 (080) 172 2601'),
(207,'+62 (002) 776-9749'),
(207,'+62-219-416-8006'),
(208,'+62 (641) 220-3103'),
(208,'+62-37-236-8314'),
(209,'(0276) 012 2777'),
(210,'+62-16-679-9840'),
(210,'+62-79-334-9605'),
(211,'(0678) 997-3080'),
(212,'(095) 814-6717'),
(213,'+62 (760) 243-7573'),
(213,'083 959 7246'),
(214,'(057) 293-4172'),
(215,'+62 (32) 971-6743'),
(215,'+62-800-545-2280'),
(216,'+62 (0550) 842 8910'),
(216,'+62 (056) 193 1837'),
(217,'(0711) 991 8081'),
(217,'+62 (13) 524-7871'),
(218,'(0548) 214-5653'),
(218,'+62 (55) 213 7603'),
(219,'+62 (072) 081-1556'),
(219,'+62 (86) 326 6678'),
(220,'(068) 455-0489'),
(220,'+62 (60) 602-9084'),
(221,'+62 (67) 389 9756'),
(222,'+62-410-856-2884'),
(223,'+62 (0346) 810 5485'),
(223,'+62-188-485-2008'),
(224,'+62 (001) 749-0753'),
(225,'0893544926'),
(226,'083 158 9357'),
(227,'+62 (026) 611-0627'),
(227,'+62-715-869-8864'),
(228,'(0677) 622-8277'),
(229,'+62 (88) 137 7028'),
(230,'+62 (15) 454 4702'),
(230,'+62 (196) 115-9408'),
(231,'+62 (96) 044 3270'),
(232,'+62 (085) 106-7949'),
(233,'(0567) 922 8777'),
(234,'+62 (51) 996 6365'),
(235,'0807412094'),
(236,'0818139890'),
(237,'083 780 1820'),
(238,'+62 (0339) 718 9569'),
(238,'+62-070-562-6085'),
(239,'(041) 118 2422'),
(240,'+62 (0124) 620 9520'),
(241,'+62-046-152-6546'),
(242,'+62 (003) 399-3788'),
(243,'(014) 057 8617'),
(243,'+62-63-048-9138'),
(244,'+62 (49) 671 8325'),
(244,'0823868700'),
(245,'(0942) 007 4946'),
(245,'+62 (398) 402-8914'),
(246,'+62 (061) 284 6398'),
(247,'+62 (020) 970 0268'),
(247,'+62 (547) 435 5379'),
(248,'(0681) 180-7156'),
(249,'+62 (620) 175-3703'),
(249,'+62-45-319-4337'),
(250,'+62 (047) 930-3614'),
(250,'087 283 9992'),
(251,'+62 (092) 124 9384'),
(252,'(064) 187-7580'),
(252,'+62-089-487-0605'),
(253,'+62 (090) 361 3325'),
(253,'+62-0046-875-0635'),
(254,'+62 (0110) 760-9543'),
(254,'+62 (073) 505-8319'),
(255,'(0179) 110 0651'),
(256,'+62 (07) 049 6110'),
(257,'+62-76-769-9150'),
(257,'0804661963'),
(258,'(0859) 311 0919'),
(259,'+62-928-456-7972'),
(260,'+62-0200-249-6854'),
(261,'+62-89-203-8806'),
(261,'0822749545'),
(262,'+62 (333) 153-7795'),
(263,'+62 (946) 521 7073'),
(263,'+62-33-793-3756'),
(264,'+62-620-501-5081'),
(265,'+62 (47) 395 0033'),
(266,'+62 (0234) 258-3230'),
(266,'+62 (41) 026-1534'),
(267,'(0355) 289 6112'),
(267,'+62 (33) 865 5963'),
(268,'+62-099-384-2530'),
(269,'+62 (0289) 731 7912'),
(269,'086 857 8725'),
(270,'+62 (877) 749-6903'),
(270,'+62-0373-634-6692'),
(271,'+62 (481) 214-7853'),
(272,'+62 (754) 602-7341'),
(272,'+62-078-956-7766'),
(273,'+62 (148) 881-4311'),
(274,'+62 (050) 835 3544'),
(275,'(090) 782 6333'),
(275,'+62 (0392) 840 4456'),
(276,'(0389) 877-5153'),
(277,'(063) 778-0529'),
(278,'+62 (40) 755 3666'),
(279,'+62 (999) 966 3618'),
(279,'0845909022'),
(280,'(037) 159 7412'),
(280,'+62 (082) 097-0231'),
(281,'(043) 348 1705'),
(282,'080 256 6212'),
(282,'0867163234'),
(283,'(072) 994-9387'),
(284,'(0779) 014-7546'),
(284,'+62 (0302) 716 9385'),
(285,'+62 (060) 324-8413'),
(285,'+62 (63) 296-7629'),
(286,'(0681) 511-8567'),
(286,'+62 (43) 807-0176'),
(287,'+62 (636) 004 1257'),
(288,'(0711) 837-2742'),
(289,'+62 (004) 275 2308'),
(289,'+62 (09) 124-1862'),
(290,'+62-063-353-3106'),
(290,'+62-708-164-7090'),
(291,'+62-05-313-1220'),
(292,'(081) 558-3938'),
(293,'(050) 637 3149'),
(293,'(061) 331 5520'),
(294,'+62 (013) 649-7189'),
(295,'(0150) 997-6838'),
(295,'+62 (41) 956-2525'),
(296,'+62-30-914-3639'),
(297,'+62 (002) 574 3263'),
(297,'+62-039-752-4508'),
(298,'(040) 355-3380'),
(298,'+62-0396-852-7017'),
(299,'+62 (030) 232 1298'),
(300,'+62-381-674-2336');
/*!40000 ALTER TABLE `teleponpengguna` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Table structure for table `tindakanperawatan`
--

DROP TABLE IF EXISTS `tindakanperawatan`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `tindakanperawatan` (
  `waktu_pelaksanaan` datetime NOT NULL,
  `nomor_catatan` int(11) NOT NULL,
  `deskripsi_tindakan` text NOT NULL,
  `biaya` decimal(12,2) NOT NULL CHECK (`biaya` >= 0),
  `id_dokter` int(11) NOT NULL,
  PRIMARY KEY (`waktu_pelaksanaan`,`nomor_catatan`),
  KEY `fk_tindakan_catmed` (`nomor_catatan`),
  KEY `fk_tindakan_dokter` (`id_dokter`),
  CONSTRAINT `fk_tindakan_catmed` FOREIGN KEY (`nomor_catatan`) REFERENCES `catatanmedis` (`nomor_catatan`) ON DELETE CASCADE ON UPDATE CASCADE,
  CONSTRAINT `fk_tindakan_dokter` FOREIGN KEY (`id_dokter`) REFERENCES `dokterhewan` (`id_dokter`) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tindakanperawatan`
--

SET @OLD_AUTOCOMMIT=@@AUTOCOMMIT, @@AUTOCOMMIT=0;
LOCK TABLES `tindakanperawatan` WRITE;
/*!40000 ALTER TABLE `tindakanperawatan` DISABLE KEYS */;
INSERT INTO `tindakanperawatan` VALUES
('2020-01-11 18:59:14',15,'Pemeriksaan darah lengkap',1140364.64,10),
('2020-01-30 08:39:20',41,'Pemeriksaan darah lengkap',1061696.60,3),
('2020-02-06 19:40:21',27,'Pemberian antibiotik',49671.07,9),
('2020-02-18 14:38:06',50,'Vaksinasi distemper',379982.78,3),
('2020-02-18 15:33:07',21,'Pemeriksaan rutin bulanan',1443167.06,19),
('2020-02-23 06:14:40',19,'Infus cairan',765768.65,14),
('2020-04-14 07:37:42',39,'Pemberian obat cacing',683500.57,2),
('2020-05-02 20:52:45',39,'Infus cairan',1313590.18,7),
('2020-05-12 04:53:21',34,'Pemeriksaan darah lengkap',363905.58,19),
('2020-05-17 10:31:47',48,'Pemberian antibiotik',618341.39,8),
('2020-05-24 15:54:43',18,'Pasang microchip',1388134.92,9),
('2020-05-25 15:55:52',32,'Infus cairan',1459128.10,13),
('2020-06-24 08:46:21',14,'Pemberian obat cacing',231699.97,10),
('2020-06-28 00:45:07',30,'Pemberian antibiotik',854519.28,14),
('2020-06-28 09:09:58',37,'Cabut gigi susu',849225.87,10),
('2020-07-06 19:29:14',24,'Pasang microchip',936306.97,20),
('2020-07-17 02:17:01',45,'Pasang microchip',823216.17,2),
('2020-08-18 22:28:37',3,'Infus cairan',1419152.46,2),
('2020-08-26 23:40:21',31,'Cabut gigi susu',557659.25,8),
('2020-08-31 15:59:32',45,'Vaksinasi rabies',763669.46,12),
('2020-09-02 15:55:07',22,'Vaksinasi distemper',1385283.50,12),
('2020-09-07 11:13:59',44,'Vaksinasi distemper',989260.19,2),
('2020-09-20 03:39:12',50,'Pasang microchip',1094144.67,18),
('2020-09-28 05:21:12',9,'Pemberian antibiotik',301402.88,15),
('2020-11-28 11:43:22',12,'Pemeriksaan rutin bulanan',1379122.05,17),
('2020-12-14 14:53:39',37,'Pasang microchip',116093.12,18),
('2020-12-16 14:18:23',13,'Grooming medis',72520.54,14),
('2021-02-20 14:40:48',34,'Pemberian obat cacing',597918.10,9),
('2021-02-24 18:18:29',50,'Perawatan luka',1154595.83,14),
('2021-02-24 22:49:53',7,'Perawatan luka',474370.54,13),
('2021-04-14 05:35:12',46,'Pemeriksaan darah lengkap',38483.66,17),
('2021-04-24 03:32:54',18,'Cabut gigi susu',585499.99,5),
('2021-06-03 11:53:14',5,'Operasi sterilisasi',666147.50,14),
('2021-06-28 02:52:37',36,'Pemberian antibiotik',948339.88,14),
('2021-08-14 19:20:21',35,'Penanganan kutu dan parasit',101016.53,13),
('2021-08-19 07:29:29',16,'Ultrasound abdomen',939176.74,9),
('2021-09-01 13:07:11',10,'Grooming medis',511551.30,13),
('2021-09-20 01:54:07',33,'Pemeriksaan rutin bulanan',869734.97,5),
('2021-09-23 02:17:00',21,'Vaksinasi rabies',571789.76,14),
('2021-10-10 23:53:56',31,'Operasi sterilisasi',1307473.93,11),
('2021-10-20 01:31:55',1,'Cabut gigi susu',1407310.65,4),
('2021-11-09 13:57:02',43,'Ultrasound abdomen',1148742.94,20),
('2021-11-17 18:08:29',34,'Vaksinasi rabies',635001.30,9),
('2021-11-18 11:23:32',37,'Operasi sterilisasi',1118347.26,1),
('2021-12-01 02:54:26',16,'Infus cairan',468586.22,5),
('2022-01-17 02:54:23',37,'Penanganan kutu dan parasit',860876.48,13),
('2022-02-07 23:50:07',15,'Operasi sterilisasi',704422.29,5),
('2022-04-10 13:10:37',23,'Perawatan luka',1197531.09,3),
('2022-04-15 12:03:05',9,'Pemeriksaan rutin bulanan',469168.62,16),
('2022-04-29 04:44:24',41,'Pemeriksaan darah lengkap',1180733.50,15),
('2022-05-29 10:25:25',50,'Penanganan kutu dan parasit',586476.07,18),
('2022-07-05 15:35:53',8,'Pemberian obat cacing',567127.26,8),
('2022-07-06 17:05:25',40,'Vaksinasi distemper',1202131.38,3),
('2022-08-16 05:37:56',23,'Pasang microchip',410803.40,17),
('2022-09-10 21:14:25',25,'Vaksinasi rabies',557032.84,3),
('2022-09-17 06:33:49',34,'Pemeriksaan rutin bulanan',1052054.90,14),
('2022-09-23 06:31:27',44,'Penanganan kutu dan parasit',374433.45,5),
('2022-10-26 05:19:39',28,'Pasang microchip',859638.53,4),
('2022-10-27 15:34:11',35,'Infus cairan',591173.17,5),
('2022-12-30 04:09:31',11,'Pemeriksaan darah lengkap',274666.92,18),
('2023-01-10 11:12:21',15,'Rontgen thorax',840730.64,13),
('2023-03-15 14:54:22',20,'Pemeriksaan darah lengkap',296221.49,18),
('2023-03-24 07:58:31',26,'Vaksinasi rabies',581325.11,16),
('2023-04-11 21:25:17',21,'Rontgen thorax',43202.41,14),
('2023-05-15 04:04:32',2,'Cabut gigi susu',400324.40,1),
('2023-05-17 07:01:46',14,'Perawatan luka',1374811.53,18),
('2023-05-17 12:19:22',39,'Vaksinasi rabies',1454157.78,11),
('2023-05-22 11:06:13',49,'Pemeriksaan rutin bulanan',1395090.54,10),
('2023-07-05 06:11:34',1,'Vaksinasi distemper',227800.01,8),
('2023-09-19 23:39:44',7,'Perawatan luka',367412.41,3),
('2023-10-13 17:54:35',20,'Vaksinasi distemper',555878.31,9),
('2023-10-25 13:21:09',25,'Rontgen thorax',1307689.92,13),
('2023-11-09 18:21:47',4,'Pemeriksaan darah lengkap',219617.90,14),
('2023-11-27 01:21:09',26,'Operasi sterilisasi',858378.22,8),
('2023-12-18 19:46:55',32,'Infus cairan',170447.14,16),
('2024-01-06 16:10:48',5,'Pemeriksaan rutin bulanan',71808.93,14),
('2024-01-25 15:49:36',9,'Perawatan luka',250948.71,20),
('2024-01-27 22:55:43',49,'Cabut gigi susu',1373800.91,20),
('2024-02-24 17:03:35',31,'Pemberian antibiotik',48204.36,18),
('2024-02-27 02:13:46',49,'Pemeriksaan darah lengkap',1450599.69,16),
('2024-02-27 17:39:19',38,'Pemeriksaan darah lengkap',159138.93,17),
('2024-03-20 09:42:32',41,'Pemeriksaan darah lengkap',900864.44,7),
('2024-03-31 06:37:00',16,'Operasi sterilisasi',837197.74,11),
('2024-04-10 06:38:30',44,'Cabut gigi susu',1386581.37,10),
('2024-05-11 02:21:32',36,'Vaksinasi distemper',626865.16,1),
('2024-06-04 15:28:38',21,'Cabut gigi susu',676714.82,7),
('2024-06-20 16:36:58',6,'Vaksinasi distemper',683953.27,12),
('2024-07-27 04:53:46',12,'Cabut gigi susu',604761.48,5),
('2024-08-02 13:08:32',6,'Vaksinasi distemper',221159.96,12),
('2024-08-11 08:39:12',12,'Pemberian antibiotik',1485727.22,2),
('2024-08-18 12:47:13',44,'Ultrasound abdomen',650588.84,13),
('2024-09-02 09:53:34',27,'Infus cairan',1371375.41,18),
('2024-09-17 21:37:26',23,'Infus cairan',162269.78,20),
('2024-10-16 18:20:27',35,'Pasang microchip',995711.13,6),
('2024-11-12 00:27:38',16,'Pemeriksaan rutin bulanan',793629.18,13),
('2024-11-24 21:54:01',47,'Infus cairan',495382.95,20),
('2024-12-01 18:00:48',14,'Perawatan luka',132718.55,14),
('2024-12-19 12:34:29',36,'Operasi sterilisasi',879235.82,14),
('2024-12-25 06:00:55',12,'Infus cairan',999491.06,3),
('2024-12-30 13:24:55',49,'Pemberian obat cacing',1093350.23,18);
/*!40000 ALTER TABLE `tindakanperawatan` ENABLE KEYS */;
UNLOCK TABLES;
COMMIT;
SET AUTOCOMMIT=@OLD_AUTOCOMMIT;

--
-- Final view structure for view `hewandenganumur`
--

/*!50001 DROP VIEW IF EXISTS `hewandenganumur`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8mb4 */;
/*!50001 SET character_set_results     = utf8mb4 */;
/*!50001 SET collation_connection      = utf8mb4_uca1400_ai_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`root`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `hewandenganumur` AS select `h`.`id_hewan` AS `id_hewan`,`h`.`nama` AS `nama`,`h`.`tgl_lahir` AS `tgl_lahir`,`h`.`jenis_kelamin` AS `jenis_kelamin`,`h`.`status` AS `status`,`h`.`id_shelter` AS `id_shelter`,timestampdiff(YEAR,`h`.`tgl_lahir`,curdate()) AS `umur` from `hewan` `h` */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

-- Dump completed on 2026-05-08 19:43:38
